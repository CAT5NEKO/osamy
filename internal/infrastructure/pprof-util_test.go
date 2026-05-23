package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewPprofMuxRoutes(t *testing.T) {
	server := httptest.NewServer(NewPprofMux())
	defer server.Close()

	cases := []string{
		"/debug/pprof/",
		"/debug/pprof/heap",
		"/debug/pprof/goroutine",
	}

	for _, path := range cases {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("request failed for %s: %v", path, err)
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			t.Fatalf("unexpected status for %s: %d", path, response.StatusCode)
		}
		response.Body.Close()
	}
}
