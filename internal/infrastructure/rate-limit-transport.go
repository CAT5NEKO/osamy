package infrastructure

import (
	"net/http"
)

type RateLimitAwareTransport struct {
	transport   http.RoundTripper
	onRateLimit func(host string)
}

func NewRateLimitAwareTransport(transport http.RoundTripper, onRateLimit func(host string)) *RateLimitAwareTransport {
	return &RateLimitAwareTransport{
		transport:   transport,
		onRateLimit: onRateLimit,
	}
}

func (t *RateLimitAwareTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.transport.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode == http.StatusTooManyRequests && t.onRateLimit != nil {
		t.onRateLimit(request.URL.Host)
	}
	return response, nil
}
