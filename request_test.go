package remilia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestOptions(t *testing.T) {
	t.Run("Successful build with valid options", func(t *testing.T) {
		req, err := NewRequest(
			WithMethod("GET"),
			WithURL("http://example.com"),
			WithHeader("Content-Type", "application/json"),
			WithHeader("Accept", "application/xml"),
			WithBody([]byte(`{"foo":"bar"}`)),
			WithQueryParam("foo", "bar"),
		)

		assert.NoError(t, err, "NewRequest should not return error")
		assert.Equal(t, "GET", req.Method, "Method should be GET")
		assert.Equal(t, "http://example.com", req.URL, "URL should be http://example.com")
		assert.Equal(t, "application/json", req.Headers["Content-Type"], "Content-Type should be application/json")
		assert.Equal(t, "application/xml", req.Headers["Accept"], "Accept should be application/xml")
		assert.Equal(t, []byte(`{"foo":"bar"}`), req.Body, "Body should be {\"foo\":\"bar\"}")
		assert.Equal(t, "bar", req.QueryParams["foo"], "QueryParams should be bar")
	})

	t.Run("Failed build with invalid options", func(t *testing.T) {
		// TODO: add more test for options
		req, err := NewRequest(
			WithMethod("INVALID"),
		)

		assert.Error(t, err, "NewRequest should return error")
		assert.Nil(t, req, "Request should be nil")
	})
}
