package remilia

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

type Request struct {
	Method      []byte
	URL         []byte
	Headers     *fasthttp.Args
	Body        []byte
	QueryParams *fasthttp.Args
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
		req.URL = append(req.URL[:0], url...)
		return nil
	}
}

func withHeader(key, value string) requestOption {
	return func(req *Request) error {
		req.Headers.Add(key, value)
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
		req.QueryParams.Add(key, value)
		return nil
	}
}

func newRequest(opts ...requestOption) (*Request, error) {
	req := &Request{
		Headers:     fasthttp.AcquireArgs(),
		QueryParams: fasthttp.AcquireArgs(),
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
	fasthttpReq.SetRequestURIBytes(req.URL)

	req.Headers.VisitAll(func(key, value []byte) {
		fasthttpReq.Header.SetBytesKV(key, value)
	})

	fasthttpReq.BodyWriter().Write(req.Body)

	args := fasthttpReq.URI().QueryArgs()
	req.QueryParams.VisitAll(func(key, value []byte) {
		args.AddBytesKV(key, value)
	})

	return fasthttpReq
}
