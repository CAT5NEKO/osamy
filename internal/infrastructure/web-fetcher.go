package infrastructure

import (
	"context"
	"io"
	"math/rand"
	"net/http"
)

const MaxFetchResponseBodySize = 10 * 1024 * 1024

type WebFetcher struct {
	httpClient *http.Client
	userAgents []string
}

func NewWebFetcher(httpClient *http.Client) *WebFetcher {
	return &WebFetcher{
		httpClient: httpClient,
		userAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		},
	}
}

func (fetcher *WebFetcher) Fetch(ctx context.Context, url string) (*http.Response, error) {
	request, requestError := http.NewRequestWithContext(ctx, "GET", url, nil)
	if requestError != nil {
		return nil, requestError
	}

	userAgent := fetcher.userAgents[rand.Intn(len(fetcher.userAgents))]
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	request.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")

	response, fetchError := fetcher.httpClient.Do(request)
	if fetchError != nil {
		return nil, fetchError
	}

	response.Body = io.NopCloser(io.LimitReader(response.Body, MaxFetchResponseBodySize))

	return response, nil
}

func (fetcher *WebFetcher) FetchAsBot(ctx context.Context, url string) (*http.Response, error) {
	request, requestError := http.NewRequestWithContext(ctx, "GET", url, nil)
	if requestError != nil {
		return nil, requestError
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	request.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")

	response, fetchError := fetcher.httpClient.Do(request)
	if fetchError != nil {
		return nil, fetchError
	}

	response.Body = io.NopCloser(io.LimitReader(response.Body, MaxFetchResponseBodySize))

	return response, nil
}
