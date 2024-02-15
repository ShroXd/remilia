package remilia

import (
	"log"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/text/encoding/simplifiedchinese"
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

	client     httpClient
	logger     Logger
	urlMatcher func(s string) bool
}

func New() (*Remilia, error) {
	return newCurried(WithDefaultClient(), WithDefaultLogger())()
}

func newCurried(opts ...remiliaOption) func() (*Remilia, error) {
	return func() (*Remilia, error) {
		r := &Remilia{}

		for _, opt := range opts {
			opt(r)
		}

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

		if r.client == nil {
			internalClient := newFastHTTPClient()
			client, err := newClient(
				withInternalClient(internalClient),
				withDocumentCreator(&defaultDocumentCreator{}),
				withClientLogger(r.logger),
			)
			if err != nil {
				log.Printf("Error: Failed to create instance of the struct due to: %v", err)
			}
			r.client = client
		}
		r.urlMatcher = urlMatcher()
		return r, nil
	}
}

type remiliaOption func(*Remilia)

func WithClient(client httpClient) remiliaOption {
	return func(r *Remilia) {
		r.client = client
	}
}

// TODO: export the option fns for the client
func WithDefaultClient() remiliaOption {
	return func(r *Remilia) {
		internalClient := newFastHTTPClient()
		client, err := newClient(
			withInternalClient(internalClient),
			withDocumentCreator(&defaultDocumentCreator{}),
			withClientLogger(r.logger),
			withTransformer(simplifiedchinese.GBK.NewDecoder()),
		)
		if err != nil {
			log.Printf("Error: Failed to create instance of the struct due to: %v", err)
		}
		r.client = client
	}
}

func WithLogger(logger Logger) remiliaOption {
	return func(r *Remilia) {
		r.logger = logger
	}
}

func WithDefaultLogger() remiliaOption {
	return func(r *Remilia) {
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
		req, err := newRequest(withURL(urlStr))
		if err != nil {
			return err
		}
		put(req)
		return nil
	}
}

func (r *Remilia) unitWrappedFunc(fn func(in *goquery.Document, put Put[string])) stageFunc[*Request] {
	return func(get Get[*Request], put Put[*Request], inCh chan *Request) error {
		wrappedPut := func(in string) {
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

		worker := func(done <-chan struct{}, requests <-chan *Request) <-chan *Response {
			responses := make(chan *Response)
			go func() {
				defer close(responses)
				for req := range requests {
					select {
					case <-done:
						return
					default:
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

		done := make(chan struct{})
		defer close(done)

		var workers []<-chan *Response

		for i := 0; i < 1; i++ {
			workers = append(workers, worker(done, inCh))
		}

		mergedResponses := fanIn(done, workers...)

		for resp := range mergedResponses {
			fn(resp.document, wrappedPut)
		}

		return nil
	}
}

func (r *Remilia) Just(urlStr string) processorDef[*Request] {
	return newProcessor[*Request](r.justWrappedFunc(urlStr))
}

func (r *Remilia) Unit(fn func(in *goquery.Document, put Put[string])) stageDef[*Request] {
	return newStage[*Request](r.unitWrappedFunc(fn))
}

func (r *Remilia) Do(producerDef processorDef[*Request], stageDefs ...stageDef[*Request]) error {
	pipeline, err := newPipeline[*Request](producerDef, stageDefs...)
	if err != nil {
		return err
	}

	return pipeline.execute()
}
