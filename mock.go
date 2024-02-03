package remilia

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/valyala/fasthttp"
)

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, context ...LogContext) {
	m.Called(msg, context)
}

func (m *MockLogger) Info(msg string, context ...LogContext) {
	m.Called(msg, context)
}

func (m *MockLogger) Warn(msg string, context ...LogContext) {
	m.Called(msg, context)
}

func (m *MockLogger) Error(msg string, context ...LogContext) {
	m.Called(msg, context)
}

func (m *MockLogger) Panic(msg string, context ...LogContext) {
	m.Called(msg, context)
}

func newMockLogger(t *testing.T) *MockLogger {
	mockLogger := new(MockLogger)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Error", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Panic", mock.Anything, mock.Anything).Return(nil)

	return mockLogger
}

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
