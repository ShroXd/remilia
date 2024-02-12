package remilia

import (
	"io"
	"os"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/mock"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type FileSystemOperations interface {
	MkdirAll(path string, perm os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
}

type FileSystem struct{}

func (fs FileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs FileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return os.OpenFile(name, flag, perm)
}

type MockFileSystem struct {
	MkdirAllErr  error
	OpenFileErr  error
	OpenFileMock *os.File
}

func (mfs MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return mfs.MkdirAllErr
}

func (mfs MockFileSystem) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	return mfs.OpenFileMock, mfs.OpenFileErr
}

type DocumentCreator interface {
	NewDocumentFromReader(io.Reader) (*goquery.Document, error)
}

type DefaultDocumentCreator struct{}

func (d DefaultDocumentCreator) NewDocumentFromReader(r io.Reader) (*goquery.Document, error) {
	return goquery.NewDocumentFromReader(r)
}

type MockDocumentCreator struct {
	Doc *goquery.Document
	Err error
}

func (d MockDocumentCreator) NewDocumentFromReader(r io.Reader) (*goquery.Document, error) {
	return d.Doc, d.Err
}

type LogContext map[string]interface{}
type Logger interface {
	Debug(msg string, context ...LogContext)
	Info(msg string, context ...LogContext)
	Warn(msg string, context ...LogContext)
	Error(msg string, context ...LogContext)
	Panic(msg string, context ...LogContext)
}

type DefaultLogger struct {
	internal *zap.Logger
}

func (l *DefaultLogger) Debug(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Debug(msg, fields...)
}

func (l *DefaultLogger) Info(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext((context)))
	l.internal.Info(msg, fields...)
}

func (l *DefaultLogger) Warn(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Warn(msg, fields...)
}

func (l *DefaultLogger) Error(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Error(msg, fields...)
}

func (l *DefaultLogger) Panic(msg string, context ...LogContext) {
	fields := convertToZapFields(getContext(context))
	l.internal.Panic(msg, fields...)
}

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

type InternalClient interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
}

type MockInternalClient struct {
	mock.Mock
}

func (m *MockInternalClient) Do(req *fasthttp.Request, resp *fasthttp.Response) error {
	args := m.Called(req, resp)
	return args.Error(0)
}

type HTTPClient interface {
	Execute(request []*Request) (*Response, error)
}

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Execute(request []*Request) (*Response, error) {
	args := m.Called(request)
	return args.Get(0).(*Response), args.Error(1)
}
