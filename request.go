package remilia

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

type Request struct {
	Method      []byte
	URL         string
	Headers     map[string]string
	Body        []byte
	QueryParams map[string]string
}

type requestOption func(*Request) error

func withMethod(method string) requestOption {
	return func(req *Request) error {
		if method == "GET" || method == "POST" || method == "PUT" || method == "DELETE" {
			req.Method = append(req.Method[:0], method...)
			return nil
		} else {
			return fmt.Errorf("invalid method: %s", method)
		}
	}
}

func withURL(url string) requestOption {
	return func(req *Request) error {
		req.URL = url
		return nil
	}
}

func withHeader(key, value string) requestOption {
	return func(req *Request) error {
		req.Headers[key] = value
		return nil
	}
}

func withBody(body []byte) requestOption {
	return func(req *Request) error {
		req.Body = body
		return nil
	}
}

func withQueryParam(key, value string) requestOption {
	return func(req *Request) error {
		req.QueryParams[key] = value
		return nil
	}
}

func newRequest(opts ...requestOption) (*Request, error) {
	req := &Request{
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
	}

	for _, opt := range opts {
		err := opt(req)
		if err != nil {
			return nil, err
		}
	}

	return req, nil
}

// func emptyRequest() *Request {
// 	// TODO: re-use the existing request insteace
// 	return &Request{}
// }

func (req *Request) build() *fasthttp.Request {
	fasthttpReq := fasthttp.AcquireRequest()

	fasthttpReq.Header.SetMethodBytes(req.Method)
	fasthttpReq.SetRequestURI(req.URL)

	for k, v := range req.Headers {
		fasthttpReq.Header.Set(k, v)
	}

	fasthttpReq.BodyWriter().Write(req.Body)

	args := fasthttpReq.URI().QueryArgs()
	for k, v := range req.QueryParams {
		args.Add(k, v)
	}

	return fasthttpReq
}
