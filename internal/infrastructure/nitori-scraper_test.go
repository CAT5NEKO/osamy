package infrastructure

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNitoriScraperResolvesRelativeThumbnail(t *testing.T) {
	fixture := loadFixture(t, "nitori_relative_image.html")

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(writer, fixture)
	}))
	defer server.Close()

	scraper := NewNitoriScraper(newTestWebFetcher())
	summary, err := scraper.Scrape(context.Background(), server.URL+"/home")
	if err != nil {
		t.Fatalf("scrape failed: %v", err)
	}
	if summary == nil {
		t.Fatalf("summary is nil")
	}
	expected := server.URL + "/images/nitori.jpg"
	if summary.Thumbnail != expected {
		t.Fatalf("unexpected thumbnail: got %s want %s", summary.Thumbnail, expected)
	}
}
