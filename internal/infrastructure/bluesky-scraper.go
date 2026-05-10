package infrastructure

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/osamy/internal/domain"
)

type BlueskyScraper struct {
	webFetcher *WebFetcher
}

func NewBlueskyScraper(webFetcher *WebFetcher) *BlueskyScraper {
	return &BlueskyScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *BlueskyScraper) CanHandle(targetUrl string) bool {
	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return false
	}
	hostname := strings.ToLower(parsedUrl.Hostname())
	return hostname == "bsky.app" || hostname == "www.bsky.app"
}

func (scraper *BlueskyScraper) Scrape(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, fetchError := scraper.webFetcher.Fetch(scrapeCtx, targetUrl)
	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	document, parseError := BuildDocumentFromResponse(response)
	if parseError != nil {
		return nil, parseError
	}

	pageSummary := domain.NewPageSummary(targetUrl)

	title := scraper.extractMeta(document, "property", "og:title")
	if title == "" {
		title = document.Find("title").Text()
	}
	pageSummary.SetTitle(title)

	description := scraper.extractMeta(document, "property", "og:description")
	if description == "" {
		description = scraper.extractMeta(document, "name", "description")
	}
	pageSummary.SetDescription(description)

	thumbnail := scraper.extractMeta(document, "property", "og:image")
	pageSummary.SetThumbnail(ResolveRelativeUrl(targetUrl, thumbnail))

	pageSummary.SetSiteName("Bluesky")
	pageSummary.SetIcon("https://bsky.app/static/favicon-32x32.png")

	videoUrl := scraper.extractMeta(document, "property", "og:video:url")
	if videoUrl == "" {
		videoUrl = scraper.extractMeta(document, "property", "og:video")
	}
	pageSummary.SetPlayer(videoUrl, 0, 0)

	pageSummary.Finalize()
	return pageSummary, nil
}

func (scraper *BlueskyScraper) extractMeta(document *goquery.Document, attributeName, attributeValue string) string {
	selection := document.Find("meta[" + attributeName + "=\"" + attributeValue + "\"]")
	return selection.AttrOr("content", "")
}
