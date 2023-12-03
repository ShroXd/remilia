package remilia

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"

	"github.com/PuerkitoBio/goquery"
)

type Remilia struct {
	ID               string
	URL              string
	Name             string
	ConcurrentNumber int

	client          *Client
	steps           []*Stage
	wg              *sync.WaitGroup
	logger          Logger
	consoleLogLevel LogLevel
	fileLogLevel    LogLevel
}

func New(client *Client, steps ...*Stage) *Remilia {
	r := &Remilia{
		client: client,
		steps:  steps,
		wg:     &sync.WaitGroup{},
	}

	return r.init()
}

func C() *Client {
	return NewClient()
}

// init setup private deps
func (r *Remilia) init() *Remilia {
	logConfig := &LoggerConfig{
		ID:   GetOrDefault(&r.ID, uuid.NewString()),
		Name: GetOrDefault(&r.Name, "defaultName"),
		// ConsoleLevel: r.consoleLogLevel,
		// FileLevel:    r.fileLogLevel,
		ConsoleLevel: DebugLevel,
		FileLevel:    DebugLevel,
	}

	var err error
	r.logger, err = createLogger(logConfig)
	if err != nil {
		log.Printf("Error: Failed to create instance of the struct due to: %v", err)
		// TODO: consider is it necessary to stop entire application?
	}

	if r.client == nil {
		r.client = NewClient()
	}
	r.client.SetLogger(r.logger)

	return r
}

func (r *Remilia) fetch(in <-chan *Request) <-chan *goquery.Document {
	r.logger.Debug("fetch", LogContext{
		"remilia": r,
	})
	out := make(chan *goquery.Document)

	r.wg.Add(1)
	go func() {
		defer func() {
			close(out)
			r.wg.Done()
			r.logger.Debug("fetch is done", LogContext{
				"remilia": r,
			})
		}()

		for url := range in {
			resp, err := r.client.Execute(url)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
				// handle error
			}
			out <- resp.document
		}
	}()

	return out
}

func (r *Remilia) applyHTMLProcessing(processFunc HTMLParser, in <-chan *goquery.Document) <-chan interface{} {
	r.logger.Debug("applyHTMLProcessing", LogContext{
		"remilia": r,
	})
	out := make(chan interface{})

	r.wg.Add(1)
	go func() {
		defer func() {
			close(out)
			r.wg.Done()
			r.logger.Debug("applyHTMLProcessing is done", LogContext{
				"remilia": r,
			})
		}()

		for resp := range in {
			result := processFunc(resp)
			out <- result
		}
	}()

	return out
}

func (r *Remilia) applyURLProcessing(processFunc URLParser, in <-chan *goquery.Document) <-chan *Request {
	r.logger.Debug("applyURLProcessing", LogContext{
		"remilia": r,
	})
	out := make(chan *Request)

	r.wg.Add(1)
	go func() {
		defer func() {
			close(out)
			r.wg.Done()
			r.logger.Debug("applyURLProcessing is done", LogContext{
				"remilia": r,
			})
		}()

		for resp := range in {
			result := processFunc(resp)
			// TODO: handle the none url from user defined function
			if result == "" {
				continue
			}
			req, err := NewRequest(result)
			if err != nil {
				r.logger.Error("Failed to parse url string to *url.URL", LogContext{
					"url": result,
					"err": err,
				})
			}
			out <- req

			r.logger.Debug("applyURLProcessing get the request", LogContext{
				"result": result,
			})
		}
	}()

	return out
}

func (r *Remilia) runStage(ctx context.Context, stage *Stage, inRequestStream <-chan *Request) (<-chan *Request, <-chan interface{}) {
	fetchOutput := r.fetch(inRequestStream)
	out1, out2 := Tee(ctx, fetchOutput, &sync.WaitGroup{})

	var outRequestStream <-chan *Request

	if stage.urlGenerator != nil {
		r.logger.Debug("runStage with urlGenerator", LogContext{
			"remilia": r,
		})
		outRequestStream = r.applyURLProcessing(stage.urlGenerator, out1)
	} else {
		// TODO: find elegant way to handle the case that the stage doesn't have urlGenerator
		go func() {
			for range out1 {
			}
		}()
	}
	htmlStream := r.applyHTMLProcessing(stage.htmlProcessor, out2)

	return outRequestStream, htmlStream
}

func (r *Remilia) chainStages(ctx context.Context, requestStream <-chan *Request) {
	for _, stage := range r.steps {
		var htmlStream <-chan interface{}
		requestStream, htmlStream = r.runStage(ctx, stage, requestStream)

		r.wg.Add(1)
		go func(currentStage *Stage) {
			defer r.wg.Done()
			currentStage.dataConsumer(htmlStream)
		}(stage)
	}
}

func (r *Remilia) StreamUrls(ctx context.Context, urls []string) <-chan *Request {
	r.logger.Debug("StreamUrls", LogContext{
		"remilia": r,
	})

	out := make(chan *Request)

	r.wg.Add(1)
	go func() {
		defer func() {
			close(out)
			r.wg.Done()
			r.logger.Debug("StreamUrls is done", LogContext{
				"remilia": r,
			})
		}()

		for _, urlString := range urls {
			req, err := NewRequest(urlString)
			if err != nil {
				r.logger.Error("Failed to parse url string to *url.URL", LogContext{
					"url": urlString,
					"err": err,
				})
				continue
			}

			select {
			case <-ctx.Done():
				return
			case out <- req:
			}
		}
	}()

	return out
}

func (r *Remilia) Wait() {
	r.wg.Wait()
}

func (r *Remilia) Process(initUrl string, ctx context.Context) {
	// TODO: should return the error channel
	urls := r.StreamUrls(ctx, []string{initUrl})
	r.chainStages(ctx, urls)
}
