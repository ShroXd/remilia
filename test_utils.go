package remilia

import (
	"net/http"
	"net/http/httptest"
)

func createTestServer(fn func(rw http.ResponseWriter, req *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(fn))
}
