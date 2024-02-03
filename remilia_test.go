package remilia

import (
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TODO: spilt the internal test and the external test

func TestNew(t *testing.T) {
	instance, err := New()

	assert.NoError(t, err, "New() should not return an error")
	assert.NotNil(t, instance, "New() should return a non-nil instance")
	assert.NotNil(t, instance.client, "New() should return an instance with a non-nil client")
	assert.NotNil(t, instance.logger, "New() should return an instance with a non-nil logger")
	assert.NotNil(t, instance.urlMatcher, "New() should return an instance with a non-nil urlMatcher")
}

// TODO: update the unit test after finishing the implementation
func TestNewFastHTTPClient(t *testing.T) {
	client := newFastHTTPClient()

	assert.NotNil(t, client, "newFastHTTPClient() should return a non-nil client")
	assert.Equal(t, 10*time.Second, client.ReadTimeout, "newFastHTTPClient() should return a client with a 10s read timeout")
	assert.Equal(t, 10*time.Second, client.WriteTimeout, "newFastHTTPClient() should return a client with a 10s write timeout")
	assert.True(t, client.NoDefaultUserAgentHeader, "newFastHTTPClient() should return a client with no default user agent header")
	assert.NotNil(t, client.Dial, "newFastHTTPClient() should return a client with a non-nil dial function")
}

func TestJustWrappedFunc(t *testing.T) {
	instance := &Remilia{}
	urlStr := "http://example.com"
	justFunc := instance.justWrappedFunc(urlStr)

	requests := make([]*Request, 0)
	mockPut := func(req *Request) {
		requests = append(requests, req)
	}

	err := justFunc(nil, mockPut, nil)
	assert.NoError(t, err, "justFunc should not return an error")

	assert.Len(t, requests, 1, "justFunc should only put 1 request")
	assert.Equal(t, urlStr, requests[0].URL, "justFunc should put the correct request")
}

func setupWrappedFuncTest(t *testing.T) (*Remilia, *observer.ObservedLogs) {
	mockClient := new(MockHTTPClient)
	mockClient.On("Execute", mock.Anything).Return(&Response{
		document: &goquery.Document{},
	}, nil)

	core, recorded := observer.New(zap.DebugLevel)
	zapLogger := zap.New(core)
	logger := &DefaultLogger{internal: zapLogger}

	instance := &Remilia{
		client: mockClient,
		logger: logger,
	}

	return instance, recorded
}

func TestRelayWrappedFunc(t *testing.T) {
	t.Run("Successful execute", func(t *testing.T) {
		requests := make([]*Request, 0)
		mockPut := func(req *Request) {
			requests = append(requests, req)
		}

		mockGet := func() func() (*Request, bool) {
			firstCall := true
			return func() (*Request, bool) {
				if firstCall {
					firstCall = false
					return &Request{}, true
				} else {
					return nil, false
				}
			}
		}()

		instance, _ := setupWrappedFuncTest(t)
		instance.urlMatcher = func(s string) bool {
			return true
		}
		fn := func(in *goquery.Document, put Put[string]) {
			put("www.example.com")
		}
		relayFunc := instance.relayWrappedFunc(fn)

		err := relayFunc(mockGet, mockPut, nil)

		assert.NoError(t, err, "relayFunc should not return an error")
		assert.Len(t, requests, 1, "relayFunc should only put 1 request")
		assert.Equal(t, "www.example.com", requests[0].URL, "relayFunc should put the correct request")
	})

	t.Run("Failed to execute with invalid url", func(t *testing.T) {
		requests := make([]*Request, 0)
		mockPut := func(req *Request) {
			requests = append(requests, req)
		}

		mockGet := func() func() (*Request, bool) {
			firstCall := true
			return func() (*Request, bool) {
				if firstCall {
					firstCall = false
					return &Request{}, true
				} else {
					return nil, false
				}
			}
		}()

		instance, recorded := setupWrappedFuncTest(t)
		instance.urlMatcher = func(s string) bool {
			return false
		}
		fn := func(in *goquery.Document, put Put[string]) {
			put("invalid url")
		}
		relayFunc := instance.relayWrappedFunc(fn)

		err := relayFunc(mockGet, mockPut, nil)

		assert.NoError(t, err, "relayFunc should not return an error")
		assert.Len(t, requests, 0, "relayFunc should not put any request")

		entries := recorded.All()
		assert.Equal(t, 1, len(entries), "Expected one log entry to be recorded")
		assert.Equal(t, zap.ErrorLevel, entries[0].Level, "Incorrect log level")
		assert.Equal(t, "Failed to match url", entries[0].Message, "Incorrect message")
	})
}

// TODO: rewrite the logic about execute and url checking
// After it, optimize this unit test
func TestSinkWrappedFunc(t *testing.T) {
	instance, _ := setupWrappedFuncTest(t)
	fn := func(in *goquery.Document) error {
		return nil
	}
	sinkFunc := instance.sinkWrappedFunc(fn)

	req, err := sinkFunc(&Request{})

	assert.Equal(t, EmptyRequest(), req, "sinkFunc should return an empty request")
	assert.NoError(t, err, "sinkFunc should not return an error")
}
