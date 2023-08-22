package remilia

import (
	"net/http"
	"net/url"
	"remilia/pkg/concurrency"
	"remilia/pkg/logger"
	"remilia/pkg/network"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

type Remilia struct {
	URL              string
	Name             string
	ConcurrentNumber int

	// Limit rules
	Delay          time.Duration
	AllowedDomains []string
	UserAgent      string

	client        *network.Client
	parseCallback *ParseCallbackContainer
	pipeline      []*PipelineContainer
}

type (
	ParseCallback          func(node string)
	PipelineFn             func(node string) *url.URL
	ParseCallbackContainer struct {
		Fn       ParseCallback
		Selector string
	}

	PipelineContainer struct {
		Fn       PipelineFn
		Selector string
	}
)

func New(url string, options ...Option) *Remilia {
	r := &Remilia{
		URL:              url,
		ConcurrentNumber: 10,
	}

	r.init()

	return r.withOptions(options...)
}

// withOptions apply options to the shallow copy of current Remilia
func (r *Remilia) withOptions(opts ...Option) *Remilia {
	c := r.clone()
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

// init setup private deps
func (r *Remilia) init() {
	r.client = network.NewClient()
}

func (r *Remilia) visit(
	done <-chan struct{},
	url string,
	out chan<- *goquery.Document,
	bodyParser func(resp *http.Response) *goquery.Document,
) {
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("Request error", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	// visit function is the writer of out channel
	// thus it has responsiblity to close the channel
	// TODO: encapsulate the channels creating and close in a single function
	out <- bodyParser(resp)
	defer close(out)

	select {
	case <-done:
		return
	default:
	}
}

func (r *Remilia) responseParser(resp *http.Response) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.Error("Failed to parse html", zap.Error(err))
	}

	return doc
}

// clone function returns a shallow copy of Remilia object
func (r *Remilia) clone() *Remilia {
	copy := *r
	return &copy
}

func (r *Remilia) simpleVisit(url *url.URL) *goquery.Document {
	logger.Debug("Visiting the url", zap.String("url", url.String()))
	resp, err := http.Get(url.String())
	if err != nil {
		logger.Error("Request error", zap.Error(err))
		// TODO: pass error to caller
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.Error("Error during pass response", zap.Error(err))
	}

	return doc
}

//func (r *Remilia) testPipelineBlock(input <-chan *url.URL, selector string, callback func(d *goquery.Document) *url.URL) {
//	logger.Debug("Pipeline block start working")
//	done := make(chan struct{})
//
//	output := concurrency.FanOut(
//	done,
//	input,
//	r.ConcurrentNumber,
//	r.simpleVisit,
//	)
//
//	for doc := range output {
//	doc.Find(".pagelink a").Each(func(index int, s *goquery.Selection) {
//	href, _ := s.Attr("href")
//	logger.Debug("get url successfully", zap.String("url: ", href))
//	})
//	}
//}

func (r *Remilia) streamGenerator(urls []string) <-chan *url.URL {
	logger.Info("Transform url string list to read-only channel")
	out := make(chan *url.URL)

	go func() {
		defer close(out)
		for _, urlString := range urls {
			parsedURL, err := url.Parse(urlString)
			if err != nil {
				logger.Error("Error during parsing url", zap.Error(err))
			}

			logger.Debug("Push url to head channel", zap.String("channel", "head"), zap.String("function", "streamGenerator"), zap.String("url", urlString))
			out <- parsedURL
		}
	}()

	return out
}

// 1. pull urls from provided pool and send request.
// 2. The urls array provided by user, convert it to channel first.
func (r *Remilia) FirstGenerator() <-chan *url.URL {
	urls := []string{"https://www.23qb.net/lightnovel/"}

	urlStream := r.streamGenerator(urls)

	firstLevelURL := make(chan *goquery.Document)
	channels := make([]<-chan *goquery.Document, len(urls))
	for i := 0; i < len(urls); i++ {
		channels[i] = firstLevelURL
	}

	done := make(chan struct{})
	for start_url := range urlStream {
		go r.visit(done, start_url.String(), firstLevelURL, r.responseParser)
	}

	result := concurrency.FanIn(done, channels...)
	nextInput := make(chan *url.URL)

	// This should be the end consumer of data
	for doc := range result {
		go func(d *goquery.Document) {
			defer close(nextInput)

			d.Find(".pagelink a").Each(func(index int, s *goquery.Selection) {
				href, _ := s.Attr("href")
				url, err := url.Parse(href)
				if err != nil {
					logger.Error("Wrong url", zap.Error(err))
				}
				logger.Debug("Get url for next level pipeline", zap.String("url", url.String()), zap.Int("index", index))
				nextInput <- url
			})
		}(doc)
	}

	return nextInput
}

// User will provide this function
func (r *Remilia) simpleVisitWrapper(currentURL *url.URL) []*url.URL {
	doc := r.simpleVisit(currentURL)

	out := make([]*url.URL, 0, 5)

	logger.Debug("Parse document object", zap.String("function", "simpleVisitWrapper"))
	doc.Find(".pagelink a").Each(func(index int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		url, err := url.Parse(href)
		if err != nil {
			logger.Error("Wrong url", zap.Error(err))
		}
		logger.Debug("Get url for next level pipeline", zap.String("url", url.String()), zap.Int("index", index))
		out = append(out, url)
	})

	// TODO: convert url list to stream channel
	return out
}

// Start starts web collecting work via sending a request
func (r *Remilia) Start() error {
	// nextInput := r.FirstGenerator()

	urls := []string{"https://www.23qb.net/lightnovel/"}

	urlStream := r.streamGenerator(urls)

	done := make(chan struct{})
	nextInput := concurrency.FanOut(
		done,
		urlStream,
		r.ConcurrentNumber,
		r.simpleVisitWrapper,
	)

	for v := range nextInput {
		logger.Debug("url from channel", zap.String("url", v.String()))
	}

	// TODO: tee-channel pattern, one is for url builder, another is for content parser

	// r.testPipelineBlock(nextInput, ".pagelink", func(d *goquery.Document) *url.URL {
	//	url, _ := url.Parse("www.google.com")
	//	return url
	//	})
	return nil
}

func (r *Remilia) Parse(selector string, fn ParseCallback) {
	container := &ParseCallbackContainer{
		Selector: selector,
		Fn:       fn,
	}

	r.parseCallback = container
}

// TODO: make the return value can only call pipeline
// TODO: check the result of callback, if it's a URL, use it in the pipeline
func (r *Remilia) URLPipeline(selector string, generator PipelineFn) *Remilia {
	container := &PipelineContainer{
		Selector: selector,
		Fn:       generator,
	}

	if r.pipeline == nil {
		r.pipeline = []*PipelineContainer{container}
	} else {
		r.pipeline = append(r.pipeline, container)
	}

	return r
}
