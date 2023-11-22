package remilia

import (
	"net/http"
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

// TODO: unit test for SetHeader

func TestSetTimeout(t *testing.T) {
	client := NewClient().SetTimeout(10 * time.Second)
	assert.Equal(t, 10*time.Second, client.Timeout, "Timeout should be set correctly")
}

func TestSetProxy(t *testing.T) {
	client := NewClient().SetProxy("http://localhost:8080")

	transport, ok := client.internal.Transport.(*http.Transport)
	assert.True(t, ok, "Transport should be of type *http.Transport")
	assert.NotNil(t, transport, "Transport should not be nil")

	dummyReq, err := http.NewRequest("GET", "http://example.com", nil)
	assert.NoError(t, err, "Error creating dummy request")

	proxyURL, err := transport.Proxy(dummyReq)
	assert.NoError(t, err, "Error getting proxy URL")
	assert.NotNil(t, proxyURL, "Proxy should not be nil")

	assert.Equal(t, "http://localhost:8080", proxyURL.String(), "Proxy should be set correctly")
}
