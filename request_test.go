package remilia

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest_Visit(t *testing.T) {
	mockLogger := newMockLogger(t)

	// Create a new test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the request URL matches the expected URL
		assert.Equal(t, "http://example.com", r.URL.String())

		// Write a response to the client
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><head><title>Test Page</title></head><body><h1>Hello, World!</h1></body></html>`))
	}))
	defer ts.Close()

	// Create a new Request object
	req := &Request{
		internal: httptest.NewRequest(http.MethodGet, ts.URL, nil),
		logger:   mockLogger,
	}

	// Make a request to the test server
	err := req.Visit("http://example.com")
	assert.NoError(t, err)

	// Check that the current middleware is nil
	assert.Nil(t, req.currentMiddleware)

	// Check that the request was successful
	// TODO: use internal client for request
	//assert.Equal(t, http.StatusOK, req.internal.Response.StatusCode)
}

// TODO: unit test for processURLsConcurrently
//func TestRequest_processURLsConcurrently(t *testing.T) {
//	mockLogger := newMockLogger(t)
//
//	// Create a new test server
//	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		// Check that the request URL matches the expected URL
//		assert.Equal(t, "http://example.com", r.URL.String())
//
//		// Write a response to the client
//		w.WriteHeader(http.StatusOK)
//		w.Write([]byte(`<html><head><title>Test Page</title></head><body><a href="http://example.com"></a><h1>Hello, World!</h1></body></html>`))
//	}))
//	defer ts.Close()
//
//	// Create a new Request object
//	req := &Request{
//		internal: httptest.NewRequest(http.MethodGet, ts.URL, nil),
//		logger:   mockLogger,
//	}
//
//	// Create a URLGenerator and HTMLProcessor
//	urlGen := URLGenerator{
//		Fn: func(s *goquery.Selection) *url.URL {
//			href, exists := s.Attr("href")
//			if !exists {
//				return nil
//			}
//			u, err := url.Parse(href)
//			if err != nil {
//				return nil
//			}
//			return u
//		},
//		Selector: "a",
//	}
//	htmlProc := HTMLProcessor{
//		Fn: func(s *goquery.Selection) interface{} {
//			return s.Text()
//		},
//		Selector:     "h1",
//		DataConsumer: func(data <-chan interface{}) {},
//	}
//
//	// Create a channel of URLs to process
//	urls := []string{"http://example.com"}
//	urlStream := req.urlsToChannel(urls)
//
//	// Process the URLs concurrently
//	urlChan, _ := req.processURLsConcurrently(urlStream, urlGen, htmlProc)
//
//	// Check that the URL channel contains the expected URL
//	expectedURL, _ := url.Parse("http://example.com")
//	assert.Equal(t, expectedURL, <-urlChan)
//
//	// Check that the HTML channel contains the expected text
//	// assert.Equal(t, "Hello, World!", <-htmlChan)
//}
