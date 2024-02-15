package remilia

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestOptions(t *testing.T) {
	t.Run("WithMethod", func(t *testing.T) {
		validMethods := []string{"GET", "POST", "PUT", "DELETE"}
		for _, method := range validMethods {
			req := &Request{}
			err := withMethod(method)(req)

			assert.NoError(t, err, "WithMethod should not return error")
			assert.Equal(t, method, req.Method, "Method should be %s", method)
		}

		invalidMethod := "INVALID"
		req := &Request{}
		err := withMethod(invalidMethod)(req)

		assert.Error(t, err, "WithMethod should return error")
	})

	t.Run("WithURL", func(t *testing.T) {
		url := "http://example.com"
		req := &Request{}
		err := withURL(url)(req)

		assert.NoError(t, err, "WithURL should not return error")
		assert.Equal(t, url, req.URL, "URL should be %s", url)
	})

	t.Run("WithHeader", func(t *testing.T) {
		key, value := "Content-Type", "application/json"
		req := &Request{
			Headers: make(map[string]string),
		}
		err := withHeader(key, value)(req)

		assert.NoError(t, err, "WithHeader should not return error")
		assert.Equal(t, value, req.Headers[key], "Header should be %s", value)
	})

	t.Run("WithBody", func(t *testing.T) {
		body := []byte(`{"foo":"bar"}`)
		req := &Request{}
		err := withBody(body)(req)

		assert.NoError(t, err, "WithBody should not return error")
		assert.Equal(t, body, req.Body, "Body should be %s", body)
	})

	t.Run("WithQueryParam", func(t *testing.T) {
		key, value := "param1", "value1"
		req := &Request{
			QueryParams: make(map[string]string),
		}
		err := withQueryParam(key, value)(req)

		assert.NoError(t, err, "WithQueryParam should not return error")
		assert.Equal(t, value, req.QueryParams[key], "QueryParam should be %s", value)
	})
}

func TestNewRequest(t *testing.T) {
	req, err := newRequest(withMethod("GET"), withURL("http://example.com"), withHeader("Content-Type", "application/json"), withBody([]byte(`{"foo":"bar"}`)), withQueryParam("param1", "value1"))
	assert.NoError(t, err, "NewRequest should not return error")
	assert.Equal(t, "GET", req.Method, "Method should be GET")
	assert.Equal(t, "http://example.com", req.URL, "URL should be http://example.com")
	assert.Equal(t, "application/json", req.Headers["Content-Type"], "Header should be application/json")

	_, err = newRequest(withMethod("INVALID"))
	assert.Error(t, err, "NewRequest should return error")
}

func TestBuild(t *testing.T) {
	req, err := newRequest(withMethod("GET"), withURL("http://example.com"), withHeader("Content-Type", "application/json"), withBody([]byte(`{"foo":"bar"}`)), withQueryParam("param1", "value1"))
	assert.NoError(t, err, "NewRequest should not return error")

	fasthttpReq := req.build()
	assert.Equal(t, "GET", string(fasthttpReq.Header.Method()), "Method should be GET")
	assert.Equal(t, "http://example.com", string(fasthttpReq.Header.RequestURI()), "URL should be http://example.com")
	assert.Equal(t, "application/json", string(fasthttpReq.Header.Peek("Content-Type")), "Header should be application/json")
	assert.Equal(t, []byte(`{"foo":"bar"}`), fasthttpReq.Body(), "Body should be %s", []byte(`{"foo":"bar"}`))
}
