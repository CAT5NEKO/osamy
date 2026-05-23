package infrastructure

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func loadFixture(t *testing.T, name string) string {
	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return string(content)
}

func newTestWebFetcher() *WebFetcher {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	return NewWebFetcher(client)
}

func TestGeneralScraperImageExtractionVariants(t *testing.T) {
	fixtures := []struct {
		name     string
		expected func(base string) string
	}{
		{name: "og_image_secure.html", expected: func(_ string) string { return "https://cdn.example.com/secure.jpg" }},
		{name: "og_image_name.html", expected: func(_ string) string { return "https://cdn.example.com/og-name.jpg" }},
		{name: "twitter_image_src.html", expected: func(_ string) string { return "https://cdn.example.com/twitter-src.jpg" }},
		{name: "jsonld_image.html", expected: func(_ string) string { return "https://cdn.example.com/jsonld.jpg" }},
		{name: "itemprop_image.html", expected: func(base string) string { return base + "/images/itemprop.jpg" }},
		{name: "link_image_src.html", expected: func(_ string) string { return "https://cdn.example.com/link-src.jpg" }},
		{name: "thumbnail_name.html", expected: func(_ string) string { return "https://cdn.example.com/thumbnail.jpg" }},
	}

	fixtureBodies := map[string]string{}
	for _, fixture := range fixtures {
		fixtureBodies[fixture.name] = loadFixture(t, fixture.name)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		name := strings.TrimPrefix(request.URL.Path, "/")
		body, ok := fixtureBodies[name]
		if !ok {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(writer, body)
	}))
	defer server.Close()

	scraper := NewGeneralScraper(newTestWebFetcher())

	for _, fixture := range fixtures {
		targetURL := server.URL + "/" + fixture.name
		summary, err := scraper.Scrape(context.Background(), targetURL)
		if err != nil {
			t.Fatalf("scrape failed for %s: %v", fixture.name, err)
		}
		if summary == nil {
			t.Fatalf("summary is nil for %s", fixture.name)
		}
		expected := fixture.expected(server.URL)
		if summary.Thumbnail != expected {
			t.Fatalf("unexpected thumbnail for %s: got %s want %s", fixture.name, summary.Thumbnail, expected)
		}
	}
}

func TestGeneralScraperIconFallbacks(t *testing.T) {
	fixtures := []struct {
		name     string
		expected func(base string) string
	}{
		{name: "icon_apple_touch.html", expected: func(base string) string { return base + "/icons/apple-touch.png" }},
		{name: "icon_rel_contains.html", expected: func(_ string) string { return "https://cdn.example.com/favicon.ico" }},
	}

	fixtureBodies := map[string]string{}
	for _, fixture := range fixtures {
		fixtureBodies[fixture.name] = loadFixture(t, fixture.name)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		name := strings.TrimPrefix(request.URL.Path, "/")
		body, ok := fixtureBodies[name]
		if !ok {
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(writer, body)
	}))
	defer server.Close()

	scraper := NewGeneralScraper(newTestWebFetcher())

	for _, fixture := range fixtures {
		targetURL := server.URL + "/" + fixture.name
		summary, err := scraper.Scrape(context.Background(), targetURL)
		if err != nil {
			t.Fatalf("scrape failed for %s: %v", fixture.name, err)
		}
		if summary == nil {
			t.Fatalf("summary is nil for %s", fixture.name)
		}
		expected := fixture.expected(server.URL)
		if summary.Icon != expected {
			t.Fatalf("unexpected icon for %s: got %s want %s", fixture.name, summary.Icon, expected)
		}
	}
}

func TestGeneralScraperRetriesWithBotUserAgent(t *testing.T) {
	fixture := loadFixture(t, "bot_retry.html")
	var botRequests int32

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userAgent := request.Header.Get("User-Agent")
		if strings.Contains(userAgent, "Discordbot") {
			atomic.AddInt32(&botRequests, 1)
			writer.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(writer, fixture)
			return
		}
		writer.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	scraper := NewGeneralScraper(newTestWebFetcher())
	summary, err := scraper.Scrape(context.Background(), server.URL+"/blocked")
	if err != nil {
		t.Fatalf("scrape failed: %v", err)
	}
	if summary == nil {
		t.Fatalf("summary is nil")
	}
	if summary.Thumbnail != "https://cdn.example.com/bot.jpg" {
		t.Fatalf("unexpected thumbnail: got %s", summary.Thumbnail)
	}
	if atomic.LoadInt32(&botRequests) == 0 {
		t.Fatalf("expected bot retry request")
	}
}
