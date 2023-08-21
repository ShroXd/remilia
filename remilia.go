package remilia

import (
	"fmt"
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
	fmt.Println("Visiting: ", url)
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

func (r *Remilia) testPipelineBlock(input <-chan *url.URL, selector string, callback func(d *goquery.Document) *url.URL) {
	fmt.Println("TEST PIPELINE")
	done := make(chan struct{})

	output := concurrency.FanOut(
		done,
		input,
		r.ConcurrentNumber,
		r.simpleVisit,
	)

	for doc := range output {
		doc.Find(".pagelink a").Each(func(index int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			text := s.Text()
			fmt.Printf("INNTERRRRRRRRRRRR   Link %d: %s (%s)\n", index+1, text, href)
		})
	}
}

// TODO: call callback use parsing result
func (r *Remilia) consumerController() {}

// Start starts web collecting work via sending a request
func (r *Remilia) Start() error {
	done := make(chan struct{})
	channels := make([]<-chan *goquery.Document, r.ConcurrentNumber)

	for i := 0; i < r.ConcurrentNumber; i++ {
		ch := make(chan *goquery.Document)
		channels[i] = ch

		go r.visit(done, r.URL, ch, r.responseParser)
	}

	result := concurrency.FanIn(done, channels...)
	nextInput := make(chan *url.URL)

	for doc := range result {
		go func(d *goquery.Document) {
			defer close(nextInput)

			d.Find(".pagelink a").Each(func(index int, s *goquery.Selection) {
				href, _ := s.Attr("href")
				fmt.Printf("Link %d: (%s)\n", index+1, href)
				url, err := url.Parse(href)
				if err != nil {
					logger.Error("Wrong url", zap.Error(err))
				}
				nextInput <- url
			})
		}(doc)
	}

	// for u := range nextInput {
	//fmt.Println("Next url is: ", u)
	//}

	r.testPipelineBlock(nextInput, ".pagelink", func(d *goquery.Document) *url.URL {
		url, _ := url.Parse("www.google.com")
		return url
	})
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
