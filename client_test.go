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

type mockInternalClient struct {
	mock.Mock
}

func (m *mockInternalClient) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	args := m.Called(req, resp)
	return args.Error(0)
}

func TestNewClient(t *testing.T) {
	t.Run("Successful build", func(t *testing.T) {
		client, err := newClient(
			withInternalClient(new(mockInternalClient)),
			withDocumentCreator(&defaultDocumentCreator{}),
		)

		assert.NotNil(t, client, "NewClient should not return nil")
		assert.NoError(t, err, "NewClient should not return error")
	})

	t.Run("Successfully run buildClientOptions with valid options", func(t *testing.T) {
		client, err := newClient(
			withInternalClient(new(mockInternalClient)),
			withDocumentCreator(&defaultDocumentCreator{}),
			withBaseURL("http://example.com"),
			withHeaders(map[string]string{
				"Content-Type": "application/json",
				"Accept":       "application/xml",
			}),
			withTimeout(10*time.Second),
			withClientLogger(&defaultLogger{}),
		)

		assert.NoError(t, err, "buildClientOptions should not return error")
		assert.Equal(t, "http://example.com", client.baseURL, "BaseURL should be http://example.com")
		assert.Equal(t, "application/json", client.header.Get("Content-Type"), "Content-Type should be application/json")
		assert.Equal(t, "application/xml", client.header.Get("Accept"), "Accept should be application/xml")
		assert.Equal(t, 10*time.Second, client.timeout, "Timeout should be 10 seconds")
		assert.Equal(t, &defaultLogger{}, client.logger, "Logger should be set correctly")
	})

	t.Run("Successfully run buildClientOptions with valid hooks", func(t *testing.T) {
		mockPreRequestHook := func(client *backendClient, req *Request) error {
			return nil
		}
		mockPostRequestHook := func(client *backendClient, resp *Response) error {
			return nil
		}
		client, err := newClient(
			withInternalClient(new(mockInternalClient)),
			withDocumentCreator(&defaultDocumentCreator{}),
			withPreRequestHooks(mockPreRequestHook, mockPreRequestHook),
			withPostResponseHooks(mockPostRequestHook, mockPostRequestHook),
		)

		assert.NoError(t, err, "buildClientOptions should not return error")
		assert.Len(t, client.udPreRequestHooks, 2, "PreRequestHooks should have 2 hooks")
		assert.Len(t, client.udPostResponseHooks, 2, "PostResponseHooks should have 2 hooks")
	})

	t.Run("Failed to run buildClientOptions with invalid options", func(t *testing.T) {
		client, err := newClient(
			withInternalClient(new(mockInternalClient)),
			withDocumentCreator(&defaultDocumentCreator{}),
			withTimeout(-1),
		)

		assert.Nil(t, client, "Options should be nil")
		assert.Error(t, err, "buildClientOptions should return error")
		assert.Equal(t, errInvalidTimeout, err, "Error should be ErrInvalidTimeout")
	})

	t.Run("Successful build with valid options", func(t *testing.T) {
		client, err := newClient(
			withInternalClient(new(mockInternalClient)),
			withDocumentCreator(&defaultDocumentCreator{}),
			withTimeout(10*time.Second),
		)

		assert.NotNil(t, client, "NewClient should not return nil")
		assert.NoError(t, err, "NewClient should not return error")
		assert.Equal(t, 10*time.Second, client.timeout, "Timeout should be 10 seconds")
	})

	t.Run("Failed to run NewClient with invalid options", func(t *testing.T) {
		client, err := newClient(
			withInternalClient(new(mockInternalClient)),
			withDocumentCreator(&defaultDocumentCreator{}),
			withTimeout(-1),
		)

		assert.Nil(t, client, "Client should be nil")
		assert.Error(t, err, "NewClient should return error")
		assert.Equal(t, errInvalidTimeout, err, "Error should be ErrInvalidTimeout")
	})
}

func setupClient(t *testing.T, hooks ...clientOptionFunc) (*backendClient, *mockInternalClient) {
	httpClient := new(mockInternalClient)
	opts := append(hooks, withInternalClient(httpClient), withDocumentCreator(&defaultDocumentCreator{}))
	client, err := newClient(opts...)
	assert.NoError(t, err)
	return client, httpClient
}

func assertExecuteSuccess(t *testing.T, client *backendClient, httpClient *mockInternalClient, setupMock func(*mockInternalClient)) {
	if setupMock != nil {
		setupMock(httpClient)
	}

	request := &Request{}
	response, err := client.execute(request)

	assert.NoError(t, err)
	assert.NotNil(t, response)

	httpClient.AssertExpectations(t)
}

func assertExecuteFailure(t *testing.T, client *backendClient, httpClient *mockInternalClient, expectedError string, setupMock func(*mockInternalClient)) {
	if setupMock != nil {
		setupMock(httpClient)
	}

	request := &Request{}
	response, err := client.execute(request)

	assert.Nil(t, response)
	assert.Error(t, err)
	assert.Equal(t, expectedError, err.Error())

	httpClient.AssertExpectations(t)
}

type mockDocumentCreator struct {
	Doc *goquery.Document
	Err error
}

func (d mockDocumentCreator) NewDocumentFromReader(r io.Reader) (*goquery.Document, error) {
	return d.Doc, d.Err
}

func TestExecute(t *testing.T) {
	t.Run("Successful execute", func(t *testing.T) {
		client, httpClient := setupClient(t)
		assertExecuteSuccess(t, client, httpClient, func(httpClient *mockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				resp := args.Get(1).(*fasthttp.Response)
				resp.SetBody([]byte("mock response"))
			}).Return(nil)
		})
	})

	t.Run("Failed to send request", func(t *testing.T) {
		core, recorded := observer.New(zap.DebugLevel)
		zapLogger := zap.New(core)
		logger := &defaultLogger{internal: zapLogger}

		client, httpClient := setupClient(
			t,
			withClientLogger(logger),
			withBackoffPool(newPool[*exponentialBackoff](
				newExponentialBackoffFactory(
					withMinDelay(1*time.Nanosecond),
					withMaxDelay(10*time.Nanosecond),
					withMultiplier(2.0),
					withMaxAttempt(1),
				))))
		httpClient.On("Do", mock.Anything, mock.Anything).Return(errors.New("test network error"))

		request := &Request{}
		response, err := client.execute(request)

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
		logger := &defaultLogger{internal: zapLogger}

		httpClient := new(mockInternalClient)
		// TODO: figure difference between such mock struct and On
		docCreator := &mockDocumentCreator{
			Doc: nil,
			Err: errors.New("test document error"),
		}
		client, err := newClient(
			withInternalClient(httpClient),
			withDocumentCreator(docCreator),
			withClientLogger(logger),
		)
		assert.NoError(t, err)

		httpClient.On("Do", mock.Anything, mock.Anything).Return(nil)

		request := &Request{}
		response, err := client.execute(request)

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
		client, httpClient := setupClient(t, withPreRequestHooks(func(client *backendClient, req *Request) error {
			req.Method = "GET"
			return nil
		}))
		assertExecuteSuccess(t, client, httpClient, func(httpClient *mockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(0).(*fasthttp.Request)
				assert.Equal(t, "GET", string(req.Header.Method()))
			}).Return(nil)
		})
	})

	t.Run("Failed execution due to pre-request hooks returning error", func(t *testing.T) {
		client, httpClient := setupClient(t, withPreRequestHooks(func(client *backendClient, req *Request) error {
			return errors.New("pre-request error")
		}))
		assertExecuteFailure(t, client, httpClient, "pre-request error", nil)
	})

	t.Run("Successful execution with post-response hooks", func(t *testing.T) {
		client, httpClient := setupClient(t, withPostResponseHooks(func(client *backendClient, resp *Response) error {
			return nil
		}))
		assertExecuteSuccess(t, client, httpClient, func(httpClient *mockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(0).(*fasthttp.Request)
				assert.Equal(t, "GET", string(req.Header.Method()), "Method should be GET")
			}).Return(nil)
		})
	})

	t.Run("Failed execution due to post-response hooks returning error", func(t *testing.T) {
		client, httpClient := setupClient(t, withPostResponseHooks(func(client *backendClient, resp *Response) error {
			return errors.New("post-response error")
		}))
		assertExecuteFailure(t, client, httpClient, "post-response error", func(httpClient *mockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Return(nil)
		})
	})

	t.Run("Successful execution with internal pre-request hooks", func(t *testing.T) {
		client, httpClient := setupClient(t, withInternalPreRequestHooks(func(client *backendClient, req *Request) error {
			req.Method = "GET"
			return nil
		}))
		assertExecuteSuccess(t, client, httpClient, func(httpClient *mockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(0).(*fasthttp.Request)
				assert.Equal(t, "GET", string(req.Header.Method()))
			}).Return(nil)
		})
	})

	t.Run("Failed execution due to internal pre-request hooks returning error", func(t *testing.T) {
		client, httpClient := setupClient(t, withInternalPreRequestHooks(func(client *backendClient, req *Request) error {
			return errors.New("pre-request error")
		}))
		assertExecuteFailure(t, client, httpClient, "pre-request error", nil)
	})

	t.Run("Successful execution with internal post-response hooks", func(t *testing.T) {
		client, httpClient := setupClient(t, withInternalPostResponseHooks(func(client *backendClient, resp *Response) error {
			return nil
		}))
		assertExecuteSuccess(t, client, httpClient, func(httpClient *mockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
				req := args.Get(0).(*fasthttp.Request)
				assert.Equal(t, "GET", string(req.Header.Method()), "Method should be GET")
			}).Return(nil)
		})
	})

	t.Run("Failed execution due to internal post-response hooks returning error", func(t *testing.T) {
		client, httpClient := setupClient(t, withInternalPostResponseHooks(func(client *backendClient, resp *Response) error {
			return errors.New("post-response error")
		}))
		assertExecuteFailure(t, client, httpClient, "post-response error", func(httpClient *mockInternalClient) {
			httpClient.On("Do", mock.Anything, mock.Anything).Return(nil)
		})
	})
}
