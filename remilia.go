package remilia

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
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

	steps []*Stage

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

func New(client *Client, steps ...*Stage) *Remilia {
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
	step *Stage,
	done <-chan struct{},
	urlStream chan<- *Request,
	htmlStream chan<- interface{},
) {
	resp, err := r.client.Execute(request)
	if err != nil {
		r.logError("Failed to get a response", request.URL, err)
		return
	}

	doc := resp.document

	select {
	case <-done:
		return
	case htmlStream <- doc:
	}

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
	step *Stage,
) (<-chan *Request, <-chan interface{}) {
	// TODO: record visited url
	urlStream := make(chan *Request)
	htmlStream := make(chan interface{})

	fmt.Println(len(reqURLStream))

	// TODO: move this ctx to outside and use it in all of the goroutines, maybe let the user provide their own ctx
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()
	// out1, out2 := Tee(ctx, reqURLStream)

	go func() {
		defer r.logger.Info("Successfully generated Request stream")
		defer close(htmlStream)
		if step.htmlProcessor == nil {
			return
		}

		for request := range reqURLStream {
			resp, err := r.client.Execute(request)
			if err != nil {
				r.logError("Failed to get a response", request.URL, err)
				return
			}

			content := step.htmlProcessor(resp.document)

			select {
			case <-done:
				return
			case htmlStream <- content:
			}
		}
	}()

	// go func() {
	// 	defer r.logger.Info("Successfully generated HTML stream")
	// 	defer close(urlStream)
	// 	if step.urlGenerator == nil {
	// 		return
	// 	}

	// 	for request := range reqURLStream {
	// 		resp, err := r.client.Execute(request)
	// 		if err != nil {
	// 			r.logError("Failed to get a response", request.URL, err)
	// 			return
	// 		}

	// 		content, err := NewRequest(step.urlGenerator(resp.document))
	// 		if err != nil {
	// 			r.logError("Failed to generate request", request.URL, err)
	// 			return
	// 		}

	// 		select {
	// 		case <-done:
	// 			return
	// 		case urlStream <- content:
	// 		}
	// 	}
	// }()

	return urlStream, htmlStream
}

func (r *Remilia) processReqsConcurrently(input <-chan *Request, step *Stage) (<-chan *Request, <-chan interface{}) {
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

// func (r *Remilia) Process(url string) {
// 	urls := []string{url}
// 	urlStream := r.urlsToReqsStream(urls)
// 	var htmlStream <-chan interface{}

// 	for _, step := range r.steps {
// 		urlStream, htmlStream = r.processReqsConcurrently(urlStream, step)

// 		// TODO: fix the data consumer timing
// 		if step.dataConsumer != nil {
// 			go step.dataConsumer(htmlStream)
// 		}
// 	}

// 	// TODO: if we do not have blocker, the program will return before getting data from network request
// 	// for res := range urlStream {
// 	// 	fmt.Println("Get result at the end of chains: ", res)
// 	// }
// }

func (r *Remilia) fetch(in <-chan string) <-chan *goquery.Document {
	out := make(chan *goquery.Document)
	go func() {
		defer close(out)
		for url := range in {
			resp, err := http.Get(url)
			if err != nil {
				fmt.Printf("Error fetching URL %s: %v\n", url, err)
				continue
			}
			doc, err := goquery.NewDocumentFromReader(resp.Body)
			if err != nil {
				// handle error
			}
			out <- doc
		}
	}()
	return out
}

func (c *Remilia) processStageInt(processFunc HTMLParser, in <-chan *goquery.Document) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)
		for resp := range in {
			result := processFunc(resp)
			out <- result
		}
	}()
	return out
}

func (c *Remilia) processStage(processFunc URLParser, in <-chan *goquery.Document) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)
		for resp := range in {
			result := processFunc(resp)
			out <- result
		}
	}()
	return out
}

func (r *Remilia) processSingleStage(stage *Stage, in <-chan string) <-chan string {
	fetchOutput := r.fetch(in)
	out1, out2 := Tee(context.TODO(), fetchOutput)
	r.processStageInt(stage.htmlProcessor, out1)
	processOutput := r.processStage(stage.urlGenerator, out2)

	return processOutput
}

func (r *Remilia) chainStages(in <-chan string) <-chan string {
	out := in
	for _, stage := range r.steps {
		out = r.processSingleStage(stage, out)
	}

	return out
}

func (r *Remilia) Process(initUrl string) {
	urls := []string{"http://localhost:8080"}
	in := make(chan string)
	var wg sync.WaitGroup

	finalStage := r.chainStages(in)

	wg.Add(1)
	go func() {
		for _, url := range urls {
			in <- url
		}
		close(in)
		wg.Done()
	}()

	// Receive the output from the last stage
	wg.Add(1)
	go func() {
		for n := range finalStage {
			fmt.Println(n)
		}
		wg.Done()
	}()

	wg.Wait()
}
