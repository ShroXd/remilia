package remilia

import (
	"remilia/pkg/network"
	"time"
)

type Remilia struct {
	Name string

	// Limit rules
	Delay          time.Duration
	AllowedDomains []string

	client network.Client
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
