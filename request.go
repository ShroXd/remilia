package remilia

import (
	"github.com/valyala/fasthttp"
)

type Build[T any] interface {
	Build() T
}

type Option interface {
	apply(*Request)
}

type optionFunc func(*Request)

func (f optionFunc) apply(req *Request) {
	f(req)
}

type Request struct {
	URL    string
	logger Logger

	options []optionFunc
}

func NewRequest(urlString string) (*Request, error) {
	return &Request{
		URL:     urlString,
		options: []optionFunc{},
	}, nil
}

func (req *Request) Build() *fasthttp.Request {
	r := fasthttp.AcquireRequest()
	for _, f := range req.options {
		f(req)
	}

	return r
}
