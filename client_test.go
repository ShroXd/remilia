package remilia

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	assert.NotNil(t, c, "NewClient should not return nil")
}

func TestSetBaseURL(t *testing.T) {
	client := NewClient().SetBaseURL("http://example.com")
	assert.Equal(t, "http://example.com", client.BaseURL, "BaseURL should be set correctly")
}

func TestSetHeaders(t *testing.T) {
	client := NewClient()
	headers := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/xml",
	}

	client.SetHeaders(headers)

	for key, val := range headers {
		assert.Equal(t, val, client.Header.Get(key), "Header should be set correctly")
	}
}

func TestSetTimeout(t *testing.T) {
	client := NewClient().SetTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, client.Timeout, "Timeout should be set correctly")
}

// func TestSetProxy(t *testing.T) {
// 	client := NewClient().SetProxy("http://localhost:8080")

// 	transport, ok := client.internal.Transport.(*http.Transport)
// 	assert.True(t, ok, "Transport should be of type *http.Transport")
// 	assert.NotNil(t, transport, "Transport should not be nil")

// 	dummyReq, err := http.NewRequest("GET", "http://example.com", nil)
// 	assert.NoError(t, err, "Error creating dummy request")

// 	proxyURL, err := transport.Proxy(dummyReq)
// 	assert.NoError(t, err, "Error getting proxy URL")
// 	assert.NotNil(t, proxyURL, "Proxy should not be nil")

// 	assert.Equal(t, "http://localhost:8080", proxyURL.String(), "Proxy should be set correctly")
// }

func TestSetLogger(t *testing.T) {
	logger := &DefaultLogger{}
	client := NewClient().SetLogger(logger)
	assert.Equal(t, logger, client.logger, "Logger should be set correctly")
}
