package remilia

import (
	"testing"

	"github.com/stretchr/testify/mock"
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

func (m *MockLogger) Fatal(msg string, context ...LogContext) {
	m.Called(msg, context)
}

// newMockLogger sets up and returns a new MockLogger with pre-defined expectations for each log level.
func newMockLogger(t *testing.T) *MockLogger {
	mockLogger := new(MockLogger)

	// Set up expectations for the methods that will be called.
	// The mock.Anything argument is used here to indicate that any argument is acceptable.
	// If you need to set specific arguments, replace mock.Anything with the actual argument value.
	mockLogger.On("Debug", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Error", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Panic", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("Fatal", mock.Anything, mock.Anything).Return(nil)

	return mockLogger
}
