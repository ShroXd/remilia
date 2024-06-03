package remilia

import (
	"log"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type fileSystemOperations interface {
	MkdirAll(path string, perm os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
}

type fileSystem struct{}

func (fs fileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs fileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

type httpClient interface {
	execute(request *Request) (*Response, error)
}

type Remilia struct {
	ID   string
	Name string

	client             httpClient
	logger             Logger
	urlMatcher         func(s string) bool
	globalStageOptions []StageOptionFunc
}

func New(opts ...RemiliaOptionFunc) (*Remilia, error) {
	r := &Remilia{}

	if r.logger == nil {
		logConfig := &loggerConfig{
			ID:           getOrDefault(&r.ID, uuid.NewString()),
			Name:         getOrDefault(&r.Name, "defaultName"),
			ConsoleLevel: debugLevel,
			FileLevel:    debugLevel,
		}

		var err error
		r.logger, err = createLogger(logConfig, &fileSystem{})
		if err != nil {
			log.Printf("Error: Failed to create instance of the struct due to: %v", err)
		}
	}

	// TODO: should I move the url mather to client?
	r.urlMatcher = urlMatcher()

	for _, opt := range opts {
		opt(r)
	}

	if r.client == nil {
		client, err := newClient(
			withInternalClient(newFastHTTPClient()),
			withDocumentCreator(&defaultDocumentCreator{}),
			withClientLogger(r.logger),
		)
		if err != nil {
			log.Printf("Error: Failed to create instance of the struct due to: %v", err)
		}

		r.client = client
	}

	return r, nil
}

func (r *Remilia) justWrappedFunc(urlStr string) func(get Get[*Request], put Put[*Request], chew Put[*Request]) error {
	return func(get Get[*Request], put Put[*Request], chew Put[*Request]) error {
		// TODO: maybe we should put the response
		req, err := newRequest(withURL(urlStr))
		if err != nil {
			return err
		}
		put(req)
		return nil
	}
}

func (r *Remilia) createWrappedPut(put Put[*Request]) Put[string] {
	return func(in string) {
		if !r.urlMatcher(in) {
			r.logger.Error("Failed to match url", logContext{
				"url": in,
			})
			return
		}

		req, err := newRequest(withURL(in))
		if err != nil {
			r.logger.Error("Failed to create request", logContext{
				"err": err,
			})
			return
		}

		put(req)
	}
}

func (r *Remilia) worker(done <-chan struct{}, requests <-chan *Request) <-chan *Response {
	responses := make(chan *Response, 100)
	go func() {
		defer close(responses)
		for {
			select {
			case <-done:
				return
			case req, ok := <-requests:
				if !ok {
					return
				}
				resp, err := r.client.execute(req)
				if err != nil {
					continue
				}
				responses <- resp
			}
		}
	}()
	return responses
}

func (r *Remilia) createWorkers(done <-chan struct{}, requests <-chan *Request, numWorkers int) []<-chan *Response {
	workers := make([]<-chan *Response, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workers[i] = r.worker(done, requests)
	}

	return workers
}

func (r *Remilia) wrapLayerFunc(fn func(in *goquery.Document, put Put[string])) actionLayerFunc[*Request] {
	return func(get Get[*Request], put Put[*Request], inCh chan *Request) error {
		wrappedPut := r.createWrappedPut(put)

		done := make(chan struct{})
		defer close(done)

		workers := r.createWorkers(done, inCh, 1)
		mergedResponses := fanIn(done, workers...)

		for resp := range mergedResponses {
			fn(resp.document, wrappedPut)
		}

		return nil
	}
}

func (r *Remilia) URLProvider(urlStr string) providerDef[*Request] {
	return newProvider[*Request](r.justWrappedFunc(urlStr))
}

type LayerFunc func(in *goquery.Document, put Put[string])

func (r *Remilia) AddLayer(fn LayerFunc, opts ...StageOptionFunc) actionLayerDef[*Request] {
	combinedOpts := append(r.globalStageOptions, opts...)

	return newActionLayer[*Request](r.wrapLayerFunc(fn), combinedOpts...)
}

func (r *Remilia) Do(pd providerDef[*Request], stageDefs ...actionLayerDef[*Request]) error {
	pipeline, err := newPipeline[*Request](pd, stageDefs...)
	if err != nil {
		return err
	}

	return pipeline.execute()
}

func newFastHTTPClient() *fasthttp.Client {
	return &fasthttp.Client{
		ReadTimeout:              10 * time.Second,
		WriteTimeout:             10 * time.Second,
		NoDefaultUserAgentHeader: true,
		MaxConnsPerHost:          5120,
		// TODO: figure out how to set timeout for TCP connection
		// Dial: fasthttpproxy.FasthttpHTTPDialer("127.0.0.1:8888"),
	}
}

type RemiliaOptionFunc func(*Remilia)

func WithClientOptions(opts ...ClientOptionFunc) RemiliaOptionFunc {
	return func(r *Remilia) {
		client, err := newClient(
			withInternalClient(newFastHTTPClient()),
			withDocumentCreator(&defaultDocumentCreator{}),
			withClientLogger(r.logger),
		)
		if err != nil {
			log.Printf("Error: Failed to create instance of the struct due to: %v", err)
		}

		for _, opt := range opts {
			if err := opt(client); err != nil {
				log.Printf("Error: Failed to create instance of the struct due to: %v", err)
			}
		}

		r.client = client
	}
}

func WithLayerOptions(opts ...StageOptionFunc) RemiliaOptionFunc {
	return func(r *Remilia) {
		r.globalStageOptions = opts
	}
}

func WithLogger(logger Logger) RemiliaOptionFunc {
	return func(r *Remilia) {
		r.logger = logger
	}
}
