package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestRateLimitAwareTransportReports429(t *testing.T) {
	var reportCount int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	transport := NewRateLimitAwareTransport(
		http.DefaultTransport,
		func(host string) {
			atomic.AddInt32(&reportCount, 1)
		},
	)

	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if atomic.LoadInt32(&reportCount) == 0 {
		t.Fatalf("expected onRateLimit to be called for 429")
	}
}

func TestRateLimitAwareTransportDoesNotReport200(t *testing.T) {
	var reportCount int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("ok"))
	}))
	defer server.Close()

	transport := NewRateLimitAwareTransport(
		http.DefaultTransport,
		func(host string) {
			atomic.AddInt32(&reportCount, 1)
		},
	)

	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if atomic.LoadInt32(&reportCount) != 0 {
		t.Fatalf("expected onRateLimit not to be called for 200")
	}
}

func TestRateLimitAwareTransportNilCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	transport := NewRateLimitAwareTransport(http.DefaultTransport, nil)

	client := &http.Client{
		Transport: transport,
	}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
}
