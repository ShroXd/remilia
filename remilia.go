package remilia

import (
	"log"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

type Remilia struct {
	ID   string
	Name string

	pipeline   pipeline[any]
	client     *Client
	logger     Logger
	urlMatcher func(s string) bool
}

func New() *Remilia {
	r := &Remilia{
		client: NewClient(),
	}

	r.init()
	return r
}

func (r *Remilia) init() {
	logConfig := &LoggerConfig{
		ID:           GetOrDefault(&r.ID, uuid.NewString()),
		Name:         GetOrDefault(&r.Name, "defaultName"),
		ConsoleLevel: DebugLevel,
		FileLevel:    DebugLevel,
	}

	var err error
	r.logger, err = createLogger(logConfig)
	if err != nil {
		log.Printf("Error: Failed to create instance of the struct due to: %v", err)
	}

	if r.client == nil {
		r.client = NewClient()
	}
	r.client.SetLogger(r.logger)
	r.urlMatcher = URLMatcher()
}

// Note: *Request is the only things we pass in the pipeline

func (r *Remilia) Just(urlStr string) ProcessorDef[*Request] {
	producerFn := func(get Get[*Request], put Put[*Request], chew Put[*Request]) error {
		// TODO: we should put the response
		req, err := NewRequest(urlStr)
		if err != nil {
			return err
		}
		put(req)
		return nil
	}

	return NewProcessor[*Request](producerFn)
}

func (r *Remilia) Relay(fn func(in *goquery.Document, put Put[string])) ProcessorDef[*Request] {
	wrappedFn := func(get Get[*Request], put Put[*Request], chew Put[*Request]) error {
		for {
			req, ok := get()
			if !ok {
				break
			}
			resp, err := r.client.Execute(req)
			if err != nil {
				r.logger.Error("Failed to execute request", LogContext{
					"err": err,
				})
			}

			wrappedPut := func(in string) {
				if !r.urlMatcher(in) {
					r.logger.Error("Failed to match url", LogContext{
						"url": in,
					})
				}

				req, err := NewRequest(in)
				if err != nil {
					r.logger.Error("Failed to create request", LogContext{
						"err": err,
					})
				}

				put(req)
			}
			fn(resp.document, wrappedPut)
		}

		return nil
	}

	return NewProcessor[*Request](wrappedFn)
}

func (r *Remilia) Sink(fn func(in *goquery.Document) error) FlowDef[*Request] {
	wrappedFn := func(in *Request) (*Request, error) {
		resp, err := r.client.Execute(in)
		if err != nil {
			r.logger.Error("Failed to execute request", LogContext{
				"err": err,
			})
		}
		fn(resp.document)

		return EmptyRequest(), nil
	}

	return NewFlow[*Request](wrappedFn)
}

func (r *Remilia) Do(producerDef ProcessorDef[*Request], stageDefs ...ProcessorDef[*Request]) error {
	pipeline, err := newPipeline[*Request](producerDef, stageDefs...)
	if err != nil {
		return err
	}

	return pipeline.execute()
}
