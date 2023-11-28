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

// withOptions apply options to the shallow copy of current Remilia
func (r *Remilia) withOptions(opts ...Option) *Remilia {
	for _, opt := range opts {
		opt.apply(r)
	}
	return r
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

func (r *Remilia) fetch(in <-chan *Request) <-chan *goquery.Document {
	out := make(chan *goquery.Document)
	go func() {
		defer close(out)
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

func (c *Remilia) processtage(processFunc URLParser, in <-chan *goquery.Document) <-chan *Request {
	out := make(chan *Request)
	go func() {
		defer close(out)
		for resp := range in {
			result := processFunc(resp)
			req, err := NewRequest(result)
			if err != nil {
			}
			out <- req
		}
	}()
	return out
}

func (r *Remilia) processSingleStage(stage *Stage, in <-chan *Request) <-chan *Request {
	fetchOutput := r.fetch(in)
	out1, out2 := Tee(context.TODO(), fetchOutput)
	r.processStageInt(stage.htmlProcessor, out1)
	processOutput := r.processtage(stage.urlGenerator, out2)

	return processOutput
}

func (r *Remilia) chainStages(in <-chan *Request) <-chan *Request {
	out := in
	for _, stage := range r.steps {
		out = r.processSingleStage(stage, out)
	}

	return out
}

func (r *Remilia) StreamUrls(ctx context.Context, urls []string) <-chan *Request {
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

func (r *Remilia) Process(initUrl string) {
	urls := r.StreamUrls(context.TODO(), []string{initUrl})

	var wg sync.WaitGroup

	finalStage := r.chainStages(urls)

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
