package remilia

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
)

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

	client *Client

	// log
	logger          Logger
	consoleLogLevel LogLevel
	fileLogLevel    LogLevel

	steps []*Step

	chain             []Middleware
	currentMiddleware *Middleware
}

// func New(url string, options ...Option) *Remilia {
// 	r := &Remilia{
// 		URL:              url,
// 		ConcurrentNumber: 10,
// 	}

// 	return r.withOptions(options...).init()
// }

func New(client *Client, steps ...*Step) *Remilia {
	r := &Remilia{
		client: client,
		steps:  steps,
	}

	return r.init()
}

func C() *Client {
	return NewClient()
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
func (r *Remilia) init() *Remilia {
	logConfig := &LoggerConfig{
		ID:           GetOrDefault(&r.ID, uuid.NewString()),
		Name:         GetOrDefault(&r.Name, "defaultName"),
		ConsoleLevel: r.consoleLogLevel,
		FileLevel:    r.fileLogLevel,
	}

	var err error
	r.logger, err = createLogger(logConfig)
	if err != nil {
		log.Printf("Error: Failed to create instance of the struct due to: %v", err)
		// TODO: consider is it necessary to stop entire application?
	}

	if r.client == nil {
		r.client = NewClient().SetLogger(r.logger)
	}

	return r
}

// clone function returns a shallow copy of Remilia object
func (r *Remilia) clone() *Remilia {
	copy := *r
	return &copy
}

func (r *Remilia) logError(msg string, reqURL *url.URL, err error) {
	r.logger.Error(msg, LogContext{
		"url": reqURL.String(),
		"err": err,
	})
}

func (r *Remilia) fetchURL(reqURL *url.URL) (io.ReadCloser, string) {
	r.logger.Info("Sending request", LogContext{
		"url": reqURL.String(),
	})

	resp, err := http.Get(reqURL.String())
	if err != nil {
		r.logError("Failed to get a response", reqURL, err)
		return nil, ""
	}

	return resp.Body, resp.Header.Get("Content-Type")
}

func (r *Remilia) parseHTML(respBody io.ReadCloser, contentType string, reqURL *url.URL) *goquery.Document {
	r.logger.Debug("Parsing HTML content", LogContext{
		"url": reqURL.String(),
	})

	bodyReader, err := charset.NewReader(respBody, contentType)
	if err != nil {
		r.logger.Error(
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
		r.logError("Failed to parse response body", reqURL, err)
		return nil
	}

	return doc
}

func (r *Remilia) urlsToReqsStream(urls []string) <-chan *Request {
	out := make(chan *Request)

	go func() {
		defer close(out)

		for _, urlString := range urls {
			req, err := NewRequest(urlString)
			if err != nil {
				r.logger.Error("Failed to parse url string to *url.URL", LogContext{
					"url": urlString,
					"err": err,
				})
			}

			r.logger.Debug("Push url to channel", LogContext{
				"url": urlString,
			})

			out <- req
		}
	}()

	return out
}

func (r *Remilia) fetchAndProcessRequest(
	request *Request,
	step *Step,
	done <-chan struct{},
	urlStream chan<- *Request,
	htmlStream chan<- interface{},
) {
	// respBody, ct := r.fetchURL(request)
	// if respBody == nil || ct == "" {
	// 	return
	// }
	// defer respBody.Close()
	resp, err := r.client.Execute(request)
	if err != nil {
		r.logError("Failed to get a response", request.URL, err)
		return
	}

	doc := resp.document

	// Extract the content of the h1 tag
	h1Text := doc.Find("h1").First().Text()
	log.Printf("H1 Tag Content: %s\n", h1Text)

	// doc := r.parseHTML(respBody, ct, request)
	// if doc == nil {
	// 	return
	// }

	// r.logger.Debug("Parsing HTML content", LogContext{"url": request.String()})
	// doc.Find(urlGen.Selector).Each(func(index int, s *goquery.Selection) {
	// 	select {
	// 	case <-done:
	// 		return
	// 	case urlStream <- urlGen.Fn(s):
	// 	}
	// })

	// doc.Find(htmlProc.Selector).Each(func(index int, s *goquery.Selection) {
	// 	select {
	// 	case <-done:
	// 		return
	// 	case htmlStream <- htmlProc.Fn(s):
	// 	}
	// })
}

func (r *Remilia) processReqsChannel(
	done <-chan struct{},
	reqURLStream <-chan *Request,
	step *Step,
) (<-chan *Request, <-chan interface{}) {
	// TODO: record visited url
	urlStream := make(chan *Request)
	htmlStream := make(chan interface{})

	fmt.Println(len(reqURLStream))

	go func() {
		defer r.logger.Info("Successfully generated Request stream")
		defer close(urlStream)
		defer close(htmlStream)

		for request := range reqURLStream {
			r.fetchAndProcessRequest(request, step, done, urlStream, htmlStream)
		}
	}()

	return urlStream, htmlStream
}

func (r *Remilia) processReqsConcurrently(input <-chan *Request, step *Step) (<-chan *Request, <-chan interface{}) {
	// TODO: this should be the configuration of steps
	numberOfWorkers := 5
	done := make(chan struct{})
	ReqChannels := make([]<-chan *Request, numberOfWorkers)
	HTMLChannels := make([]<-chan interface{}, numberOfWorkers)

	for i := 0; i < numberOfWorkers; i++ {
		urlStream, htmlStream := r.processReqsChannel(done, input, step)
		ReqChannels[i] = urlStream
		HTMLChannels[i] = htmlStream
	}

	return FanIn(done, ReqChannels...), FanIn(done, HTMLChannels...)
}

func (r *Remilia) Process(url string) (string, error) {
	urls := []string{url}
	urlStream := r.urlsToReqsStream(urls)

	for _, step := range r.steps {
		var htmlStream <-chan interface{}
		urlStream, htmlStream = r.processReqsConcurrently(urlStream, step)

		if step.dataConsumer != nil {
			go step.dataConsumer(htmlStream)
		}
	}

	for res := range urlStream {
		fmt.Println("Get result at the end of chains: ", res)
	}

	return "good", nil
}
