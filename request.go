package remilia

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
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

type Request struct {
	internal          *http.Request
	logger            Logger
	chain             []Middleware
	currentMiddleware *Middleware
}

func (req *Request) UseURL(selector string, urlGenFn func(s *goquery.Selection) *url.URL) *Request {
	req.ensureCurrentMiddleware()
	// Check if urlGenerator is already set. If so, it's an error to set it again.
	if req.currentMiddleware.urlGenerator.Fn != nil {
		req.logger.Panic("URLGenerator is already set for this middleware")
	}

	// Initialize a new Middleware and set its URLGenerator
	newURLGenerator := URLGenerator{
		Fn:       urlGenFn,
		Selector: selector,
	}
	req.currentMiddleware.urlGenerator = newURLGenerator

	return req
}

func (req *Request) UseHTML(selector string, htmlProcFn func(s *goquery.Selection) interface{}, dataConsumer DataConsumer) *Request {
	req.ensureCurrentMiddleware()
	if req.currentMiddleware.htmlProcessor.Fn != nil {
		req.logger.Panic("HTMLProcessor is already set for this middleware")
	}

	newHTMLProcessor := HTMLProcessor{
		Fn:           htmlProcFn,
		Selector:     selector,
		DataConsumer: dataConsumer,
	}
	req.currentMiddleware.htmlProcessor = newHTMLProcessor

	return req
}

func (req *Request) End() *Request {
	if req.currentMiddleware != nil {
		req.chain = append(req.chain, *req.currentMiddleware)
		req.currentMiddleware = nil
	}

	return nil
}

func (req *Request) Visit(url string) error {
	// create middleware chain
	if req.currentMiddleware != nil {
		req.chain = append(req.chain, *req.currentMiddleware)
		req.currentMiddleware = nil
	}

	req.logger.Info("Start crawling", LogContext{"url": url})

	urls := []string{url}
	urlStream := req.urlsToChannel(urls)

	for _, mw := range req.chain {
		var htmlStream <-chan interface{}
		urlStream, htmlStream = req.processURLsConcurrently(urlStream, mw.urlGenerator, mw.htmlProcessor)

		if mw.htmlProcessor.DataConsumer != nil {
			mw.htmlProcessor.DataConsumer(htmlStream)
		}
	}

	for res := range urlStream {
		fmt.Println("Get result at the end of chains: ", res)
	}

	return nil
}

func (req *Request) ensureCurrentMiddleware() {
	if req.currentMiddleware == nil {
		req.currentMiddleware = &Middleware{}
	}
}

func (req *Request) urlsToChannel(urls []string) <-chan *url.URL {
	req.logger.Debug("Creating read-only channel holding provided urls")
	out := make(chan *url.URL)

	go func() {
		defer close(out)

		for _, urlString := range urls {
			parsedURL, err := url.Parse(urlString)
			if err != nil {
				req.logger.Error("Could not parse url", LogContext{"url": urlString, "err": err})
				continue
			}

			req.logger.Debug("Push url to channel", LogContext{
				"url": urlString,
			})
			out <- parsedURL
		}
	}()

	return out
}

func (req *Request) processURLsConcurrently(input <-chan *url.URL, urlGen URLGenerator, htmlProc HTMLProcessor) (<-chan *url.URL, <-chan interface{}) {
	numberOfWorkers := 5
	done := make(chan struct{})
	URLChannels := make([]<-chan *url.URL, numberOfWorkers)
	HTMLChannels := make([]<-chan interface{}, numberOfWorkers)

	for i := 0; i < numberOfWorkers; i++ {
		urlStream, htmlStream := req.processURLsChannel(done, input, urlGen, htmlProc)
		URLChannels[i] = urlStream
		HTMLChannels[i] = htmlStream
	}

	return FanIn(done, URLChannels...), FanIn(done, HTMLChannels...)
}

// processURLsChannel reads URLs from reqURLStream, processes them, and sends the results to another channel
func (req *Request) processURLsChannel(
	done <-chan struct{},
	reqURLStream <-chan *url.URL,
	urlGen URLGenerator,
	htmlProc HTMLProcessor,
) (<-chan *url.URL, <-chan interface{}) {
	// TODO: record visited url
	urlStream := make(chan *url.URL)
	htmlStream := make(chan interface{})

	go func() {
		defer req.logger.Info("Successfully generated URL stream")
		defer close(urlStream)
		defer close(htmlStream)

		for reqURL := range reqURLStream {
			req.fetchAndProcessURL(reqURL, urlGen, htmlProc, done, urlStream, htmlStream)
		}
	}()

	return urlStream, htmlStream
}

// fetchAndProcessURL sends a request to the given URL, parses the response, and applies the callback on the HTML content matched by the selector
func (req *Request) fetchAndProcessURL(
	reqURL *url.URL,
	urlGen URLGenerator,
	htmlProc HTMLProcessor,
	done <-chan struct{},
	urlStream chan<- *url.URL,
	htmlStream chan<- interface{},
) {
	respBody, ct := req.fetchURL(reqURL)
	if respBody == nil || ct == "" {
		return
	}
	defer respBody.Close()

	doc := req.parseHTML(respBody, ct, reqURL)
	if doc == nil {
		return
	}

	req.logger.Debug("Parsing HTML content", LogContext{"url": reqURL.String()})
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

func (req *Request) parseHTML(respBody io.ReadCloser, contentType string, reqURL *url.URL) *goquery.Document {
	req.logger.Debug("Parsing HTML content", LogContext{
		"url": reqURL.String(),
	})

	bodyReader, err := charset.NewReader(respBody, contentType)
	if err != nil {
		req.logger.Error(
			"Failed to convert response body",
			LogContext{
				"url":               reqURL.String(),
				"sourceContentType": contentType,
				"targetContentType": "utf-8",
				"err":               err,
			},
		)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(bodyReader)
	if err != nil {
		req.logger.Error("Failed to parse response body", LogContext{"request URL": reqURL, "err": err})
		return nil
	}

	return doc
}

func (req *Request) fetchURL(reqURL *url.URL) (io.ReadCloser, string) {
	req.logger.Info("Sending request", LogContext{
		"url": reqURL.String(),
	})

	resp, err := http.Get(reqURL.String())
	if err != nil {
		req.logger.Error("Failed to get a response", LogContext{"request URL": reqURL, "err": err})
		return nil, ""
	}

	return resp.Body, resp.Header.Get("Content-Type")
}
