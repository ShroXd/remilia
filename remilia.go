package remilia

import (
	"log"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
)

type Remilia[T any] struct {
	ID   string
	Name string

	pipeline pipeline[any]
	client   *Client
	logger   Logger
}

func New() *Remilia[any] {
	r := &Remilia[any]{
		client: NewClient(),
	}

	r.init()
	return r
}

func (r *Remilia[T]) init() {
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
}

func (r *Remilia[T]) Just(urlStr string) ProducerDef[*Request] {
	producerFn := func(get Get[*Request], put Put[*Request], chew Put[*Request]) error {
		// TODO: we should put the response
		req, err := NewRequest(urlStr)
		if err != nil {
			return err
		}
		put(req)
		return nil
	}

	return NewProducer[*Request](producerFn)
}

func (r *Remilia[T]) Sink(fn func(in *goquery.Document) error) StageDef[*Request] {
	wrappedFn := func(in *Request) (*Request, error) {
		resp, err := r.client.Execute(in)
		if err != nil {
			r.logger.Error("Failed to execute request", LogContext{
				"err": err,
			})
		}
		fn(resp.document)

		// TODO: temp for testing
		return NewRequest("www.google.com")
	}
	return NewStage[*Request](wrappedFn)
}

func (r *Remilia[T]) Do(producerDef ProducerDef[*Request], stageDefs ...StageDef[*Request]) error {
	pipeline, err := newPipeline[*Request](producerDef, stageDefs...)
	if err != nil {
		return err
	}

	return pipeline.execute()
}
