package remilia

import (
	"fmt"
	"net/http"
	"remilia/pkg/concurrency"
	"remilia/pkg/logger"
	"remilia/pkg/network"
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

	client *network.Client
}

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

	out <- bodyParser(resp)

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

func (r *Remilia) do(request *network.Request) *http.Response {
	req, err := request.Build()
	if err != nil {
		logger.Error("Failed to build request", zap.Error(err))
	}

	resp, err := r.client.Visit(req)
	if err != nil {
		logger.Error("Failed to send request", zap.Error(err))
	}

	return resp
}

// clone function returns a shallow copy of Remilia object
func (r *Remilia) clone() *Remilia {
	copy := *r
	return &copy
}

// Start starts web collecting work via sending a request
func (r *Remilia) Start() error {
	done := make(chan struct{})
	defer close(done)

	channels := make([]<-chan *goquery.Document, r.ConcurrentNumber)

	for i := 0; i < r.ConcurrentNumber; i++ {
		ch := make(chan *goquery.Document)
		channels[i] = ch

		go r.visit(done, r.URL, ch, r.responseParser)
	}

	result := concurrency.FanIn(done, channels...)

	testParser := func(d *goquery.Document) string {
		res := d.Find("h1").Text()
		return res
	}

	for i := 0; i < r.ConcurrentNumber; i++ {
		htmlContent := <-result
		fmt.Println("Received request code: ", testParser(htmlContent))
	}

	return nil
}
