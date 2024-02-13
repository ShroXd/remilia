package remilia

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestBuildClientOptions(t *testing.T) {
}

type MockInternalClient struct {
	mock.Mock
}

func (m *MockInternalClient) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	args := m.Called(req, resp)
	return args.Error(0)
}

func TestNewClient(t *testing.T) {
	t.Run("Successful build", func(t *testing.T) {
		client, err := NewClient(
			WithInternalClient(new(MockInternalClient)),
			WithDocumentCreator(&DefaultDocumentCreator{}),
		)

		assert.NotNil(t, client, "NewClient should not return nil")
		assert.NoError(t, err, "NewClient should not return error")
	})

	t.Run("Successfully run buildClientOptions with valid options", func(t *testing.T) {
		client, err := NewClient(
			WithInternalClient(new(MockInternalClient)),
			WithDocumentCreator(&DefaultDocumentCreator{}),
			WithBaseURL("http://example.com"),
			WithHeaders(map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/xml",
			}),
			WithTimeout(10*time.Second),
			WithClientLogger(&DefaultLogger{}),
		)

		assert.NoError(t, err, "buildClientOptions should not return error")
		assert.Equal(t, "http://example.com", client.baseURL, "BaseURL should be http://example.com")
		assert.Equal(t, "application/json", client.header.Get("Content-Type"), "Content-Type should be application/json")
		assert.Equal(t, "application/xml", client.header.Get("Accept"), "Accept should be application/xml")
		assert.Equal(t, 10*time.Second, client.timeout, "Timeout should be 10 seconds")
		assert.Equal(t, &DefaultLogger{}, client.logger, "Logger should be set correctly")
	})

	t.Run("Successfully run buildClientOptions with valid hooks", func(t *testing.T) {
		mockPreRequestHook := func(client *Client, req *Request) error {
			return nil
		}
		mockPostRequestHook := func(client *Client, resp *Response) error {
			return nil
		}
		client, err := NewClient(
			WithInternalClient(new(MockInternalClient)),
			WithDocumentCreator(&DefaultDocumentCreator{}),
			WithPreRequestHooks(mockPreRequestHook, mockPreRequestHook),
			WithPostResponseHooks(mockPostRequestHook, mockPostRequestHook),
		)

		assert.NoError(t, err, "buildClientOptions should not return error")
		assert.Len(t, client.udPreRequestHooks, 2, "PreRequestHooks should have 2 hooks")
		assert.Len(t, client.udPostResponseHooks, 2, "PostResponseHooks should have 2 hooks")
	})

	t.Run("Failed to run buildClientOptions with invalid options", func(t *testing.T) {
		client, err := NewClient(
			WithInternalClient(new(MockInternalClient)),
			WithDocumentCreator(&DefaultDocumentCreator{}),
			WithTimeout(-1),
		)

		assert.Nil(t, client, "Options should be nil")
		assert.Error(t, err, "buildClientOptions should return error")
		assert.Equal(t, ErrInvalidTimeout, err, "Error should be ErrInvalidTimeout")
	})

	t.Run("Successful build with valid options", func(t *testing.T) {
		client, err := NewClient(
			WithInternalClient(new(MockInternalClient)),
			WithDocumentCreator(&DefaultDocumentCreator{}),
			WithTimeout(10*time.Second),
		)

		assert.NotNil(t, client, "NewClient should not return nil")
		assert.NoError(t, err, "NewClient should not return error")
		assert.Equal(t, 10*time.Second, client.timeout, "Timeout should be 10 seconds")
	})

	t.Run("Failed to run NewClient with invalid options", func(t *testing.T) {
		client, err := NewClient(
			WithInternalClient(new(MockInternalClient)),
			WithDocumentCreator(&DefaultDocumentCreator{}),
			WithTimeout(-1),
		)

		assert.Nil(t, client, "Client should be nil")
		assert.Error(t, err, "NewClient should return error")
		assert.Equal(t, ErrInvalidTimeout, err, "Error should be ErrInvalidTimeout")
	})
}

func setupClient(t *testing.T, hooks ...ClientOptionFunc) (*Client, *MockInternalClient) {
	httpClient := new(MockInternalClient)
	opts := append(hooks, WithInternalClient(httpClient), WithDocumentCreator(&DefaultDocumentCreator{}))
	client, err := NewClient(opts...)
	assert.NoError(t, err)
	return client, httpClient
}

func assertExecuteSuccess(t *testing.T, client *Client, httpClient *MockInternalClient, setupMock func(*MockInternalClient)) {
	if setupMock != nil {
		setupMock(httpClient)
	}

	request := &Request{}
	response, err := client.Execute([]*Request{request})

	assert.NoError(t, err)
	assert.NotNil(t, response)

	httpClient.AssertExpectations(t)
}

func assertExecuteFailure(t *testing.T, client *Client, httpClient *MockInternalClient, expectedError string, setupMock func(*MockInternalClient)) {
	if setupMock != nil {
		setupMock(httpClient)
	}

	request := &Request{}
	response, err := client.Execute([]*Request{request})

	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err.Error())

	httpClient.AssertExpectations(t)
}

type MockDocumentCreator struct {
	Doc *goquery.Document
	Err error
}

func (d MockDocumentCreator) NewDocumentFromReader(r io.Reader) (*goquery.Document, error) {
	return d.Doc, d.Err
}

func TestExecute(t *testing.T) {
	t.Run("Successful execute", func(t *testing.T) {
		client, httpClient := setupClient(t)
		assertExecuteSuccess(t, client, httpClient, func(httpClient *MockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				resp := args.Get(1).(*fasthttp.Response)
				resp.SetBody([]byte("mock response"))
			}).Return(nil)
		})
	})

	t.Run("Failed to send request", func(t *testing.T) {
		core, recorded := observer.New(zap.DebugLevel)
		zapLogger := zap.New(core)
		logger := &DefaultLogger{internal: zapLogger}

		client, httpClient := setupClient(
			t,
			WithClientLogger(logger),
			WithBackoffPool(NewPool[*ExponentialBackoff](
				NewExponentialBackoffFactory(
					WithMinDelay(1*time.Nanosecond),
					WithMaxDelay(10*time.Nanosecond),
					WithMultiplier(2.0),
					WithMaxAttempt(1),
				))))
		httpClient.On("Do", mock.Anything, mock.Anything).Return(errors.New("test network error"))

		request := &Request{}
		response, err := client.Execute([]*Request{request})

		assert.Nil(t, response, "Response should be nil")
		assert.Error(t, err, "NewClient should return error")
		assert.Equal(t, err.Error(), "test network error")

		entries := recorded.All()
		assert.Equal(t, 1, len(entries), "Expected one log entry to be recorded")
		entry := entries[0]

		assert.Equal(t, zap.ErrorLevel, entry.Level, "Incorrect log level")
		assert.Equal(t, "Failed to execute request", entry.Message, "Incorrect message")
		assert.Equal(t, "test network error", entry.ContextMap()["err"], "Incorrect context logged")
	})

	t.Run("Failed to build goquery document", func(t *testing.T) {
		core, recorded := observer.New(zap.DebugLevel)
		zapLogger := zap.New(core)
		logger := &DefaultLogger{internal: zapLogger}

		httpClient := new(MockInternalClient)
		// TODO: figure difference between such mock struct and On
		docCreator := &MockDocumentCreator{
			Doc: nil,
			Err: errors.New("test document error"),
		}
		client, err := NewClient(
			WithInternalClient(httpClient),
			WithDocumentCreator(docCreator),
			WithClientLogger(logger),
		)
		assert.NoError(t, err)

		httpClient.On("Do", mock.Anything, mock.Anything).Return(nil)

		request := &Request{}
		response, err := client.Execute([]*Request{request})

		assert.Nil(t, response, "Response should be nil")
		assert.Error(t, err, "NewClient should return error")
		assert.Equal(t, err.Error(), "test document error")

		entries := recorded.All()
		assert.Equal(t, 1, len(entries), "Expected one log entry to be recorded")
		entry := entries[0]

		assert.Equal(t, zap.ErrorLevel, entry.Level, "Incorrect log level")
		assert.Equal(t, "Failed to build goquery document", entry.Message, "Incorrect message")
		assert.Equal(t, "test document error", entry.ContextMap()["err"], "Incorrect context logged")
	})

	t.Run("Successful execution with pre-request hooks", func(t *testing.T) {
		client, httpClient := setupClient(t, WithPreRequestHooks(func(client *Client, req *Request) error {
			req.Method = "GET"
			return nil
		}))
		assertExecuteSuccess(t, client, httpClient, func(httpClient *MockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(0).(*fasthttp.Request)
				assert.Equal(t, "GET", string(req.Header.Method()))
			}).Return(nil)
		})
	})

	t.Run("Failed execution due to pre-request hooks returning error", func(t *testing.T) {
		client, httpClient := setupClient(t, WithPreRequestHooks(func(client *Client, req *Request) error {
			return errors.New("pre-request error")
		}))
		assertExecuteFailure(t, client, httpClient, "pre-request error", nil)
	})

	t.Run("Successful execution with post-response hooks", func(t *testing.T) {
		client, httpClient := setupClient(t, WithPostResponseHooks(func(client *Client, resp *Response) error {
			return nil
		}))
		assertExecuteSuccess(t, client, httpClient, func(httpClient *MockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(0).(*fasthttp.Request)
				assert.Equal(t, "GET", string(req.Header.Method()), "Method should be GET")
			}).Return(nil)
		})
	})

	t.Run("Failed execution due to post-response hooks returning error", func(t *testing.T) {
		client, httpClient := setupClient(t, WithPostResponseHooks(func(client *Client, resp *Response) error {
			return errors.New("post-response error")
		}))
		assertExecuteFailure(t, client, httpClient, "post-response error", func(httpClient *MockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Return(nil)
		})
	})

	t.Run("Successful execution with internal pre-request hooks", func(t *testing.T) {
		client, httpClient := setupClient(t, WithInternalPreRequestHooks(func(client *Client, req *Request) error {
			req.Method = "GET"
			return nil
		}))
		assertExecuteSuccess(t, client, httpClient, func(httpClient *MockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(0).(*fasthttp.Request)
				assert.Equal(t, "GET", string(req.Header.Method()))
			}).Return(nil)
		})
	})

	t.Run("Failed execution due to internal pre-request hooks returning error", func(t *testing.T) {
		client, httpClient := setupClient(t, WithInternalPreRequestHooks(func(client *Client, req *Request) error {
			return errors.New("pre-request error")
		}))
		assertExecuteFailure(t, client, httpClient, "pre-request error", nil)
	})

	t.Run("Successful execution with internal post-response hooks", func(t *testing.T) {
		client, httpClient := setupClient(t, WithInternalPostResponseHooks(func(client *Client, resp *Response) error {
			return nil
		}))
		assertExecuteSuccess(t, client, httpClient, func(httpClient *MockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(0).(*fasthttp.Request)
				assert.Equal(t, "GET", string(req.Header.Method()), "Method should be GET")
			}).Return(nil)
		})
	})

	t.Run("Failed execution due to internal post-response hooks returning error", func(t *testing.T) {
		client, httpClient := setupClient(t, WithInternalPostResponseHooks(func(client *Client, resp *Response) error {
			return errors.New("post-response error")
		}))
		assertExecuteFailure(t, client, httpClient, "post-response error", func(httpClient *MockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Return(nil)
		})
	})
}
