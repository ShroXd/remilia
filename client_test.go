package remilia

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/valyala/fasthttp"
)

func TestClientOptions(t *testing.T) {
	t.Run("Successful build with valid options", func(t *testing.T) {
		opts, err := buildClientOptions([]ClientOptionFn{
			BaseURL("http://example.com"),
			Headers(map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/xml",
			}),
			Timeout(10 * time.Second),
			ClientLogger(&DefaultLogger{}),
		})

		assert.NoError(t, err, "buildClientOptions should not return error")
		assert.Equal(t, "http://example.com", opts.baseURL, "BaseURL should be http://example.com")
		assert.Equal(t, "application/json", opts.header.Get("Content-Type"), "Content-Type should be application/json")
		assert.Equal(t, "application/xml", opts.header.Get("Accept"), "Accept should be application/xml")
		assert.Equal(t, 10*time.Second, opts.timeout, "Timeout should be 10 seconds")
		assert.Equal(t, &DefaultLogger{}, opts.logger, "Logger should be set correctly")
	})

	t.Run("Successful build with valid hooks", func(t *testing.T) {
		mockPreRequestHook := func(client *Client, req *Request) error {
			return nil
		}
		mockPostRequestHook := func(client *Client, resp *Response) error {
			return nil
		}
		opts, err := buildClientOptions([]ClientOptionFn{
			PreRequestHooks(mockPreRequestHook, mockPreRequestHook),
			PostResponseHooks(mockPostRequestHook, mockPostRequestHook),
		})

		assert.NoError(t, err, "buildClientOptions should not return error")
		assert.Len(t, opts.udPreRequestHooks, 2, "PreRequestHooks should have 2 hooks")
		assert.Len(t, opts.udPostResponseHooks, 2, "PostResponseHooks should have 2 hooks")
	})

	t.Run("Successful execute", func(t *testing.T) {
		httpClient := new(MockInternalClient)
		client, err := NewClient(httpClient)
		assert.NoError(t, err, "NewClient should not return error")

		httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			resp := args.Get(1).(*fasthttp.Response)
			resp.SetBody([]byte("mock response"))
		}).Return(nil)

		request := &Request{}
		response, err := client.Execute(request)

		assert.NoError(t, err, "Execute should not return error")
		assert.NotNil(t, response, "Response should not be nil")

		httpClient.AssertExpectations(t)
	})

	t.Run("Successful execute with pre-request hooks", func(t *testing.T) {
		httpClient := new(MockInternalClient)
		client, err := NewClient(httpClient, PreRequestHooks(func(client *Client, req *Request) error {
			req.Method = "GET"
			return nil
		}))
		assert.NoError(t, err, "NewClient should not return error")

		httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			req := args.Get(0).(*fasthttp.Request)
			assert.Equal(t, "GET", string(req.Header.Method()), "Method should be GET")
		}).Return(nil)

		request := &Request{}
		response, err := client.Execute(request)

		assert.NoError(t, err, "Execute should not return error")
		assert.NotNil(t, response, "Response should not be nil")

		httpClient.AssertExpectations(t)
	})
}

// func TestNewClient(t *testing.T) {
// 	c := NewClient()
// 	assert.NotNil(t, c, "NewClient should not return nil")
// }

// func TestSetBaseURL(t *testing.T) {
// 	client := NewClient().SetBaseURL("http://example.com")
// 	assert.Equal(t, "http://example.com", client.BaseURL, "BaseURL should be set correctly")
// }

// func TestSetHeaders(t *testing.T) {
// 	client := NewClient()
// 	headers := map[string]string{
// 		"Content-Type": "application/json",
// 		"Accept":       "application/xml",
// 	}

// 	client.SetHeaders(headers)

// 	for key, val := range headers {
// 		assert.Equal(t, val, client.Header.Get(key), "Header should be set correctly")
// 	}
// }

// func TestSetTimeout(t *testing.T) {
// 	client := NewClient().SetTimeout(10 * time.Second)
// 	assert.Equal(t, 10*time.Second, client.Timeout, "Timeout should be set correctly")
// }

// // func TestSetProxy(t *testing.T) {
// // 	client := NewClient().SetProxy("http://localhost:8080")

// // 	transport, ok := client.internal.Transport.(*http.Transport)
// // 	assert.True(t, ok, "Transport should be of type *http.Transport")
// // 	assert.NotNil(t, transport, "Transport should not be nil")

// // 	dummyReq, err := http.NewRequest("GET", "http://example.com", nil)
// // 	assert.NoError(t, err, "Error creating dummy request")

// // 	proxyURL, err := transport.Proxy(dummyReq)
// // 	assert.NoError(t, err, "Error getting proxy URL")
// // 	assert.NotNil(t, proxyURL, "Proxy should not be nil")

// // 	assert.Equal(t, "http://localhost:8080", proxyURL.String(), "Proxy should be set correctly")
// // }

// func TestSetLogger(t *testing.T) {
// 	logger := &DefaultLogger{}
// 	client := NewClient().SetLogger(logger)
// 	assert.Equal(t, logger, client.logger, "Logger should be set correctly")
// }
