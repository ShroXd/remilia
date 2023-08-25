package remilia

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"remilia/pkg/concurrency"
	"remilia/pkg/logger"
	"remilia/pkg/network"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

type DataConsumer func(data <-chan interface{})

type (
	URLGenerator struct {
		Fn       func(s *goquery.Selection) *url.URL
		Selector string
	}

	HTMLProcessor struct {
		Fn           func(s *goquery.Selection) interface{}
		Selector     string
		DataConsumer DataConsumer
	}
)

type Middleware struct {
	urlGenerator  URLGenerator
	htmlProcessor HTMLProcessor
}

type Remilia struct {
	ID               string
	URL              string
	Name             string
	ConcurrentNumber int

	// Limit rules
	Delay          time.Duration
	AllowedDomains []string
	UserAgent      string

	client *network.Client
	logger *logger.Logger

	chain             []Middleware
	currentMiddleware *Middleware
}

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

// urlsToChannel creates a channel that sends parsed URLs from a slice of URL strings
func (r *Remilia) urlsToChannel(urls []string) <-chan *url.URL {
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

// processURLsChannel reads URLs from reqURLStream, processes them, and sends the results to another channel
func (r *Remilia) processURLsChannel(
	done <-chan struct{},
	reqURLStream <-chan *url.URL,
	urlGen URLGenerator,
	htmlProc HTMLProcessor,
) (<-chan *url.URL, <-chan interface{}) {
	// TODO: record visited url
	urlStream := make(chan *url.URL)
	htmlStream := make(chan interface{})

	go func() {
		defer r.logger.Info("Successfully generated URL stream")
		defer close(urlStream)
		defer close(htmlStream)

		for reqURL := range reqURLStream {
			r.fetchAndProcessURL(reqURL, urlGen, htmlProc, done, urlStream, htmlStream)
		}
	}()

	return urlStream, htmlStream
}

// fetchAndProcessURL sends a request to the given URL, parses the response, and applies the callback on the HTML content matched by the selector
func (r *Remilia) fetchAndProcessURL(
	reqURL *url.URL,
	urlGen URLGenerator,
	htmlProc HTMLProcessor,
	done <-chan struct{},
	urlStream chan<- *url.URL,
	htmlStream chan<- interface{},
) {
	r.logger.Info("Sending request", zap.String("url", reqURL.String()))

	resp, err := http.Get(reqURL.String())
	if err != nil {
		r.logger.Error("Failed to get a response", zap.String("url", reqURL.String()), zap.Error(err))
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		r.logger.Error("Failed to parse response body", zap.String("url", reqURL.String()), zap.Error(err))
		return
	}

	r.logger.Debug("Parsing HTML content", zap.String("url", reqURL.String()))
	doc.Find(urlGen.Selector).Each(func(index int, s *goquery.Selection) {
		select {
		case <-done:
			return
		case urlStream <- urlGen.Fn(s):
		}
	})

	doc.Find(htmlProc.Selector).Each(func(index int, s *goquery.Selection) {
		select {
		case <-done:
			return
		case htmlStream <- htmlProc.Fn(s):
		}
	})
}

// processURLsConcurrently concurrently processes URLs using the callback and fans the results into a single channel
func (r *Remilia) processURLsConcurrently(input <-chan *url.URL, urlGen URLGenerator, htmlProc HTMLProcessor) (<-chan *url.URL, <-chan interface{}) {
	numberOfWorkers := 5
	done := make(chan struct{})
	URLChannels := make([]<-chan *url.URL, numberOfWorkers)
	HTMLChannels := make([]<-chan interface{}, numberOfWorkers)

	for i := 0; i < numberOfWorkers; i++ {
		urlStream, htmlStream := r.processURLsChannel(done, input, urlGen, htmlProc)
		URLChannels[i] = urlStream
		HTMLChannels[i] = htmlStream
	}

	return concurrency.FanIn(done, URLChannels...), concurrency.FanIn(done, HTMLChannels...)
}

func (r *Remilia) ensureCurrentMiddleware() {
	if r.currentMiddleware == nil {
		r.currentMiddleware = &Middleware{}
	}
}

func (r *Remilia) UseURL(selector string, urlGenFn func(s *goquery.Selection) *url.URL) *Remilia {
	r.ensureCurrentMiddleware()
	// Check if urlGenerator is already set. If so, it's an error to set it again.
	if r.currentMiddleware.urlGenerator.Fn != nil {
		r.logger.Panic("URLGenerator is already set for this middleware")
	}

	// Initialize a new Middleware and set its URLGenerator
	newURLGenerator := URLGenerator{
		Fn:       urlGenFn,
		Selector: selector,
	}
	r.currentMiddleware.urlGenerator = newURLGenerator

	return r
}

func (r *Remilia) UseHTML(selector string, htmlProcFn func(s *goquery.Selection) interface{}, dataConsumer DataConsumer) *Remilia {
	r.ensureCurrentMiddleware()
	if r.currentMiddleware.htmlProcessor.Fn != nil {
		r.logger.Panic("HTMLProcessor is already set for this middleware")
	}

	newHTMLProcessor := HTMLProcessor{
		Fn:           htmlProcFn,
		Selector:     selector,
		DataConsumer: dataConsumer,
	}
	r.currentMiddleware.htmlProcessor = newHTMLProcessor

	return r
}

func (r *Remilia) AddToChain() *Remilia {
	if r.currentMiddleware != nil {
		r.chain = append(r.chain, *r.currentMiddleware)
		r.currentMiddleware = nil
	}

	return nil
}

// TODO: check and compress chain
// Start initiates the crawling process
func (r *Remilia) Start() error {
	r.logger.Info("Starting crawl", zap.String("url", r.URL))

	urls := []string{r.URL}
	urlStream := r.urlsToChannel(urls)

	for _, mw := range r.chain {
		var htmlStream <-chan interface{}
		urlStream, htmlStream = r.processURLsConcurrently(urlStream, mw.urlGenerator, mw.htmlProcessor)

		if mw.htmlProcessor.DataConsumer != nil {
			go mw.htmlProcessor.DataConsumer(htmlStream)
		}
	}

	for res := range urlStream {
		fmt.Println("Get result at the end of chains: ", res)
	}

	return nil
}
