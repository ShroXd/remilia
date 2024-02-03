package remilia

import (
	"github.com/stretchr/testify/mock"
	"github.com/valyala/fasthttp"
)

type MockInternalClient struct {
	mock.Mock
}

func (m *MockInternalClient) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	args := m.Called(req, resp)
	return args.Error(0)
}

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Execute(request *Request) (*Response, error) {
	args := m.Called(request)
	return args.Get(0).(*Response), args.Error(1)
}
