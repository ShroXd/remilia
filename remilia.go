package remilia

import (
	"net/http"
	"remilia/pkg/concurrency"
	"remilia/pkg/logger"
	"remilia/pkg/network"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

type Remilia struct {
	URL              string
	Name             string
	ConcurrentNumber int

	// Limit rules
	Delay          time.Duration
	AllowedDomains []string
	UserAgent      string

	client        *network.Client
	parseCallback []*ParseCallbackContainer
}

type (
	ParseCallback          func(node string)
	ParseCallbackContainer struct {
		Fn       ParseCallback
		Selector string
	}
)

func New(url string, options ...Option) *Remilia {
	r := &Remilia{
		URL:              url,
		ConcurrentNumber: 10,
	}

	r.init()

	return r.withOptions(options...)
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
func (r *Remilia) init() {
	r.client = network.NewClient()
}

func (r *Remilia) visit(
	done <-chan struct{},
	url string,
	out chan<- *goquery.Document,
	bodyParser func(resp *http.Response) *goquery.Document,
) {
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("Request error", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	// visit function is the writer of out channel
	// thus it has responsiblity to close the channel
	// TODO: encapsulate the channels creating and close in a single function
	out <- bodyParser(resp)
	defer close(out)

	select {
	case <-done:
		return
	default:
	}
}

func (r *Remilia) responseParser(resp *http.Response) *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		logger.Error("Failed to parse html", zap.Error(err))
	}

	return doc
}

// clone function returns a shallow copy of Remilia object
func (r *Remilia) clone() *Remilia {
	copy := *r
	return &copy
}

// Start starts web collecting work via sending a request
func (r *Remilia) Start() error {
	var wg sync.WaitGroup

	done := make(chan struct{})
	defer close(done)

	channels := make([]<-chan *goquery.Document, r.ConcurrentNumber)

	for i := 0; i < r.ConcurrentNumber; i++ {
		ch := make(chan *goquery.Document)
		channels[i] = ch

		go r.visit(done, r.URL, ch, r.responseParser)
	}

	result := concurrency.FanIn(done, channels...)

	// call all callback for each result from request
	wg.Add(len(r.parseCallback) * r.ConcurrentNumber)
	for doc := range result {
		for _, cb := range r.parseCallback {
			go func(document *goquery.Document, container *ParseCallbackContainer) {
				container.Fn(document.Find(container.Selector).Text())
				defer wg.Done()
			}(doc, cb)
		}
	}

	go func() {
		wg.Wait()
	}()

	return nil
}

func (r *Remilia) Parse(selector string, fn ParseCallback) {
	container := &ParseCallbackContainer{
		Selector: selector,
		Fn:       fn,
	}

	if r.parseCallback == nil {
		r.parseCallback = []*ParseCallbackContainer{container}
	} else {
		r.parseCallback = append(r.parseCallback, container)
	}
}
