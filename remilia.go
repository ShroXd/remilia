package remilia

import (
	"net/http"
	"remilia/pkg/logger"
	"remilia/pkg/network"
	"time"

	"go.uber.org/zap"
)

type Remilia struct {
	Name string

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

func New(name string, options ...Option) *Remilia {
	r := &Remilia{
		Name: name,
	}

	r.Init()

	return r.WithOptions(options...)
}

// WithOptions apply options to the shallow copy of current Remilia
func (r *Remilia) WithOptions(opts ...Option) *Remilia {
	c := r.clone()
	for _, opt := range opts {
		opt.apply(c)
	}
	return c
}

// Init setup private deps
func (r *Remilia) Init() {
	r.client = network.NewClient()
}

// Do starts web collecting work via sending a request
func (r *Remilia) Do(request *network.Request) *http.Response {
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
