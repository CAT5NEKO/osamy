package infrastructure

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/user/osamy/internal/domain"
)

const MaxRedirectCount = 5

func NewSafeHttpTransport() *http.Transport {
	safeDialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, splitError := net.SplitHostPort(address)
			if splitError != nil {
				return nil, fmt.Errorf("invalid address: %s", address)
			}

			resolvedAddresses, lookupError := net.DefaultResolver.LookupIPAddr(ctx, host)
			if lookupError != nil {
				return nil, lookupError
			}

			for _, resolvedAddress := range resolvedAddresses {
				if domain.IsPrivateIp(resolvedAddress.IP) {
					return nil, fmt.Errorf("connection to private address is blocked")
				}
			}

			return safeDialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   10,
		DisableCompression:    false,
	}
}

func NewSafeRedirectPolicy() func(request *http.Request, via []*http.Request) error {
	return func(request *http.Request, via []*http.Request) error {
		if len(via) >= MaxRedirectCount {
			return fmt.Errorf("stopped after %d redirects", MaxRedirectCount)
		}

		redirectHostname := request.URL.Hostname()
		if domain.IsPrivateHostname(redirectHostname) {
			return fmt.Errorf("redirect to private address is blocked")
		}

		scheme := request.URL.Scheme
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("redirect to non-http scheme is blocked")
		}

		return nil
	}
}
