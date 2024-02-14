package remilia

import (
	"log"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

type FileSystemOperations interface {
	MkdirAll(path string, perm os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
}

type FileSystem struct{}

func (fs FileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs FileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

type HTTPClient interface {
	Execute(request *Request) (*Response, error)
}

type Remilia struct {
	ID   string
	Name string

	client     HTTPClient
	logger     Logger
	urlMatcher func(s string) bool
}

func New() (*Remilia, error) {
	return newCurried(WithDefaultClient(), WithDefaultLogger())()
}

func newCurried(opts ...Option) func() (*Remilia, error) {
	return func() (*Remilia, error) {
		r := &Remilia{}

		for _, opt := range opts {
			opt(r)
		}

		if r.logger == nil {
			logConfig := &LoggerConfig{
				ID:           GetOrDefault(&r.ID, uuid.NewString()),
				Name:         GetOrDefault(&r.Name, "defaultName"),
				ConsoleLevel: DebugLevel,
				FileLevel:    DebugLevel,
			}

			var err error
			r.logger, err = createLogger(logConfig, &FileSystem{})
			if err != nil {
				log.Printf("Error: Failed to create instance of the struct due to: %v", err)
			}
		}

		if r.client == nil {
			internalClient := newFastHTTPClient()
			client, err := NewClient(
				WithInternalClient(internalClient),
				WithDocumentCreator(&DefaultDocumentCreator{}),
				WithClientLogger(r.logger),
			)
			if err != nil {
				log.Printf("Error: Failed to create instance of the struct due to: %v", err)
			}
			r.client = client
		}
		r.urlMatcher = URLMatcher()
		return r, nil
	}
}

type Option func(*Remilia)

func WithClient(client HTTPClient) Option {
	return func(r *Remilia) {
		r.client = client
	}
}

func WithDefaultClient() Option {
	return func(r *Remilia) {
		internalClient := newFastHTTPClient()
		client, err := NewClient(
			WithInternalClient(internalClient),
			WithDocumentCreator(&DefaultDocumentCreator{}),
			WithClientLogger(r.logger),
		)
		if err != nil {
			log.Printf("Error: Failed to create instance of the struct due to: %v", err)
		}
		r.client = client
	}
}

func WithLogger(logger Logger) Option {
	return func(r *Remilia) {
		r.logger = logger
	}
}

func WithDefaultLogger() Option {
	return func(r *Remilia) {
		logConfig := &LoggerConfig{
			ID:           GetOrDefault(&r.ID, uuid.NewString()),
			Name:         GetOrDefault(&r.Name, "defaultName"),
			ConsoleLevel: DebugLevel,
			FileLevel:    DebugLevel,
		}

		var err error
		r.logger, err = createLogger(logConfig, &FileSystem{})
		if err != nil {
			log.Printf("Error: Failed to create instance of the struct due to: %v", err)
		}
	}
}

func newFastHTTPClient() *fasthttp.Client {
	return &fasthttp.Client{
		ReadTimeout:              10 * time.Second,
		WriteTimeout:             10 * time.Second,
		NoDefaultUserAgentHeader: true,
		// TODO: figure out how to set timeout for TCP connection
		Dial: fasthttpproxy.FasthttpHTTPDialer("127.0.0.1:4780"),
	}
}

func (r *Remilia) justWrappedFunc(urlStr string) func(get Get[*Request], put Put[*Request], chew Put[*Request]) error {
	return func(get Get[*Request], put Put[*Request], chew Put[*Request]) error {
		// TODO: we should put the response
		req, err := NewRequest(WithURL(urlStr))
		if err != nil {
			return err
		}
		put(req)
		return nil
	}
}

func (r *Remilia) unitWrappedFunc(fn func(in *goquery.Document, put Put[string], chew Put[string])) StageFunc[*Request] {
	return func(get Get[*Request], put Put[*Request], chew Put[*Request], inCh chan *Request) error {
		wrappedPut := func(in string) {
			if !r.urlMatcher(in) {
				r.logger.Error("Failed to match url", LogContext{
					"url": in,
				})
				return
			}

			req, err := NewRequest(WithURL(in))
			if err != nil {
				r.logger.Error("Failed to create request", LogContext{
					"err": err,
				})
				return
			}

			put(req)
		}

		wrappedChew := func(in string) {
			if !r.urlMatcher(in) {
				r.logger.Error("Failed to match url", LogContext{
					"url": in,
				})
				return
			}

			req, err := NewRequest(WithURL(in))
			if err != nil {
				r.logger.Error("Failed to create request", LogContext{
					"err": err,
				})
				return
			}

			chew(req)
		}

		worker := func(done <-chan struct{}, requests <-chan *Request) <-chan *Response {
			responses := make(chan *Response)
			go func() {
				defer close(responses)
				for req := range requests {
					select {
					case <-done:
						return
					default:
						resp, err := r.client.Execute(req)
						if err != nil {
							continue
						}
						responses <- resp
					}
				}
			}()
			return responses
		}

		done := make(chan struct{})
		defer close(done)

		var workers []<-chan *Response

		for i := 0; i < 1; i++ {
			workers = append(workers, worker(done, inCh))
		}

		mergedResponses := FanIn(done, workers...)

		for resp := range mergedResponses {
			fn(resp.document, wrappedPut, wrappedChew)
		}

		return nil
	}
}

func (r *Remilia) Just(urlStr string) ProcessorDef[*Request] {
	return NewProcessor[*Request](r.justWrappedFunc(urlStr))
}

func (r *Remilia) Unit(fn func(in *goquery.Document, put Put[string], chew Put[string])) StageDef[*Request] {
	return NewStage[*Request](r.unitWrappedFunc(fn))
}

func (r *Remilia) Do(producerDef ProcessorDef[*Request], stageDefs ...StageDef[*Request]) error {
	pipeline, err := newPipeline[*Request](producerDef, stageDefs...)
	if err != nil {
		return err
	}

	return pipeline.execute()
}
