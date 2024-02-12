package remilia

import (
	"log"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
)

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
				internalClient,
				&DefaultDocumentCreator{},
				ClientLogger(r.logger),
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
			internalClient,
			&DefaultDocumentCreator{},
			ClientLogger(r.logger),
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

// func (r *Remilia) relayWrappedFunc(fn func(in *goquery.Document, put Put[string])) func(get Get[*Request], put Put[*Request], chew Put[*Request]) error {
// 	return func(get Get[*Request], put Put[*Request], chew Put[*Request]) error {
// 		for {
// 			req, ok := get()
// 			if !ok {
// 				break
// 			}
// 			resp, err := r.client.Execute(req)
// 			if err != nil {
// 				r.logger.Error("Failed to execute request", LogContext{
// 					"err": err,
// 				})
// 			}

// 			wrappedPut := func(in string) {
// 				if !r.urlMatcher(in) {
// 					r.logger.Error("Failed to match url", LogContext{
// 						"url": in,
// 					})
// 					return
// 				}

// 				req, err := NewRequest(WithURL(in))
// 				if err != nil {
// 					r.logger.Error("Failed to create request", LogContext{
// 						"err": err,
// 					})
// 					return
// 				}

// 				put(req)
// 			}
// 			fn(resp.document, wrappedPut)
// 		}

// 		return nil
// 	}
// }

func (r *Remilia) unitWrappedFunc(fn func(in *goquery.Document, put Put[string], chew Put[string])) StageFunc[*Request] {
	return func(get BatchGetFunc[*Request], put Put[*Request], chew Put[*Request]) error {
		reqs, err := get()
		if err != nil {
			return err
		}

		resp, err := r.client.Execute(reqs)
		if err != nil {
			r.logger.Error("Failed to execute request", LogContext{
				"err": err,
			})
			return err
		}

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

		fn(resp.document, wrappedPut, wrappedChew)

		return nil
	}
}

// func (r *Remilia) sinkWrappedFunc(fn func(in *goquery.Document) error) func(in *Request) (*Request, error) {
// 	return func(in *Request) (*Request, error) {
// 		resp, err := r.client.Execute(in)
// 		if err != nil {
// 			r.logger.Error("Failed to execute request", LogContext{
// 				"err": err,
// 			})
// 		}
// 		fn(resp.document)

// 		return EmptyRequest(), nil
// 	}
// }

func (r *Remilia) Just(urlStr string) ProcessorDef[*Request] {
	return NewProcessor[*Request](r.justWrappedFunc(urlStr))
}

// func (r *Remilia) Relay(fn func(in *goquery.Document, put Put[string])) ProcessorDef[*Request] {
// 	return NewProcessor[*Request](r.relayWrappedFunc(fn))
// }

func (r *Remilia) Unit(fn func(in *goquery.Document, put Put[string], chew Put[string])) StageDef[*Request] {
	return NewStage[*Request](r.unitWrappedFunc(fn))
}

// func (r *Remilia) Sink(fn func(in *goquery.Document) error) FlowDef[*Request] {
// 	return NewFlow[*Request](r.sinkWrappedFunc(fn))
// }

func (r *Remilia) Do(producerDef ProcessorDef[*Request], stageDefs ...StageDef[*Request]) error {
	pipeline, err := newPipeline[*Request](producerDef, stageDefs...)
	if err != nil {
		return err
	}

	return pipeline.execute()
}
