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

func TestNewRemilia(t *testing.T) {
	t.Run("With default client and logger", func(t *testing.T) {
		instance, err := New()

		assert.NoError(t, err, "newCurried() should not return an error")
		assert.NotNil(t, instance, "newCurried() should return a non-nil instance")
		assert.NotNil(t, instance.client, "newCurried() should return an instance with a non-nil client")
		assert.NotNil(t, instance.logger, "newCurried() should return an instance with a non-nil logger")
		assert.NotNil(t, instance.urlMatcher, "newCurried() should return an instance with a non-nil urlMatcher")
	})

	t.Run("Without client", func(t *testing.T) {
		instance, err := New()

		assert.NoError(t, err, "newCurried() should not return an error")
		assert.NotNil(t, instance, "newCurried() should return a non-nil instance")
		assert.NotNil(t, instance.client, "newCurried() should return an instance with a non-nil client")
		assert.NotNil(t, instance.logger, "newCurried() should return an instance with a non-nil logger")
		assert.NotNil(t, instance.urlMatcher, "newCurried() should return an instance with a non-nil urlMatcher")
	})

	t.Run("Without logger", func(t *testing.T) {
		instance, err := New()

		assert.NoError(t, err, "newCurried() should not return an error")
		assert.NotNil(t, instance, "newCurried() should return a non-nil instance")
		assert.NotNil(t, instance.client, "newCurried() should return an instance with a non-nil client")
		assert.NotNil(t, instance.logger, "newCurried() should return an instance with a non-nil logger")
		assert.NotNil(t, instance.urlMatcher, "newCurried() should return an instance with a non-nil urlMatcher")
	})
}

// TODO: update the unit test after finishing the implementation
func TestNewFastHTTPClient(t *testing.T) {
	client := newFastHTTPClient()

	assert.NotNil(t, client, "newFastHTTPClient() should return a non-nil client")
	assert.Equal(t, 10*time.Second, client.ReadTimeout, "newFastHTTPClient() should return a client with a 10s read timeout")
	assert.Equal(t, 10*time.Second, client.WriteTimeout, "newFastHTTPClient() should return a client with a 10s write timeout")
	assert.True(t, client.NoDefaultUserAgentHeader, "newFastHTTPClient() should return a client with no default user agent header")
	// assert.NotNil(t, client.Dial, "newFastHTTPClient() should return a client with a non-nil dial function")
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
	assert.Equal(t, []byte(urlStr), requests[0].URL, "justFunc should put the correct request")
}

type mockHTTPClient struct {
	mock.Mock
}

func (m *mockHTTPClient) execute(request *Request) (*Response, error) {
	args := m.Called(request)
	return args.Get(0).(*Response), args.Error(1)
}

func setupWrappedFuncTest(t *testing.T) (*Remilia, *observer.ObservedLogs) {
	mockClient := new(mockHTTPClient)
	mockClient.On("execute", mock.Anything).Return(&Response{
		document: &goquery.Document{},
	}, nil)

	core, recorded := observer.New(zap.DebugLevel)
	zapLogger := zap.New(core)
	logger := &defaultLogger{internal: zapLogger}

	instance := &Remilia{
		client: mockClient,
		logger: logger,
	}

	return instance, recorded
}

func TestDo(t *testing.T) {
	instance, _ := New()

	processorFunc := func(get Get[*Request], put, chew Put[*Request]) error {
		return nil
	}

	stageFunc := func(get Get[*Request], put Put[*Request], inCh chan *Request) error {
		return nil
	}

	err := instance.Do(newProcessor(processorFunc), newStage(stageFunc))

	assert.NoError(t, err, "Do should not return an error")
}
