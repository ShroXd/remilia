package remilia

import (
	"log"
	"net/http"
	"net/url"
	"remilia/pkg/concurrency"
	"remilia/pkg/logger"
	"remilia/pkg/network"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
	// "go.uber.org/zap"
)

type Remilia struct {
	ID               string
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
	logger        *logger.Logger
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
	logger, err := logger.NewLogger(r.ID, r.Name)
	if err != nil {
		log.Printf("Error: Failed to create instance of the struct due to: %v", err)
		// TODO: consider is it necessary to stop entire application?
	}

	r.logger = logger
	r.client = network.NewClient()
}

// clone function returns a shallow copy of Remilia object
func (r *Remilia) clone() *Remilia {
	copy := *r
	return &copy
}

func (r *Remilia) streamGenerator(urls []string) <-chan *url.URL {
	r.logger.Debug("Creating read-only channel holding provided url")
	out := make(chan *url.URL)

	go func() {
		defer close(out)

		for _, urlString := range urls {
			parsedURL, err := url.Parse(urlString)
			if err != nil {
				r.logger.Error("Failed to parse url string to *url.URL", zap.String("url", urlString), zap.Error(err))
			}

			r.logger.Debug("Push url to channel", zap.String("url", urlString))
			out <- parsedURL
		}
	}()

	return out
}

func (r *Remilia) urlGeneratorHandler(selector string, callback func(s *goquery.Selection) *url.URL) func(reqURL *url.URL) []*url.URL {
	return func(reqURL *url.URL) []*url.URL {
		r.logger.Info("Sending request", zap.String("url", reqURL.String()))

		resp, err := http.Get(reqURL.String())
		if err != nil {
			r.logger.Error("Failed to get a response", zap.String("url", reqURL.String()), zap.Error(err))
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			r.logger.Error("Failed to parse response body", zap.String("url", reqURL.String()), zap.Error(err))
		}

		out := make([]*url.URL, 0, 5)

		r.logger.Debug("Parsing HTML content")
		doc.Find(".pagelink a").Each(func(index int, s *goquery.Selection) {
			out = append(out, callback(s))
		})

		return out
	}
}

// Start starts web collecting work via sending a request
func (r *Remilia) Start() error {
	// nextInput := r.FirstGenerator()

	urls := []string{"https://www.23qb.net/lightnovel/"}

	urlStream := r.streamGenerator(urls)

	selector := ".pagelink a"
	callback := func(s *goquery.Selection) *url.URL {
		href, _ := s.Attr("href")
		url, _ := url.Parse(href)

		return url
	}

	done := make(chan struct{})
	nextInput := concurrency.FanOut(
		done,
		urlStream,
		r.ConcurrentNumber,
		r.urlGeneratorHandler(selector, callback),
	)

	thirdInput := concurrency.FanOut(
		done,
		nextInput,
		r.ConcurrentNumber,
		r.urlGeneratorHandler(selector, callback),
	)

	for v := range thirdInput {
		r.logger.Info("Get url from previous level channel", zap.String("url", v.String()))
	}

	// TODO: tee-channel pattern, one is for url builder, another is for content parser

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
