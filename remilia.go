package remilia

import (
	"fmt"
	"net/http"
	"remilia/pkg/concurrency"
	"remilia/pkg/logger"
	"remilia/pkg/network"
	"time"

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

type Option interface {
	apply(*Remilia)
}

type optionFunc func(*Remilia)

func (f optionFunc) apply(r *Remilia) {
	f(r)
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
	out chan<- interface{},
	bodyParser func(resp *http.Response) interface{},
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

func (r *Remilia) responseParser(resp *http.Response) interface{} {
	return resp.StatusCode
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

	channels := make([]<-chan interface{}, r.ConcurrentNumber)

	for i := 0; i < r.ConcurrentNumber; i++ {
		ch := make(chan interface{})
		channels[i] = ch

		go r.visit(done, r.URL, ch, r.responseParser)
	}

	result := concurrency.FanIn(done, channels...)

	for i := 0; i < r.ConcurrentNumber; i++ {
		htmlContent := <-result
		fmt.Println("Received request code: ", htmlContent)
	}

	return nil
}

// ConcurrentNumber set number of goroutines for network request
func ConcurrentNumber(num int) Option {
	return optionFunc(func(r *Remilia) {
		r.ConcurrentNumber = num
	})
}

// Name set name for scraper
func Name(name string) Option {
	return optionFunc(func(r *Remilia) {
		r.Name = name
	})
}

// AllowedDomains sets a string list that specifies the domains accessible to the web scraper for crawling
func AllowedDomains(domains ...string) Option {
	return optionFunc(func(r *Remilia) {
		r.AllowedDomains = domains
	})
}

// Delay sets sleep duration before web scraper sends request
func Delay(delay time.Duration) Option {
	return optionFunc(func(r *Remilia) {
		r.Delay = delay
	})
}

// UserAgent sets user agent used by request
func UserAgent(ua string) Option {
	return optionFunc(func(r *Remilia) {
		r.UserAgent = ua
	})
}
