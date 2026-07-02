package infrastructure

import (
	"context"
	"time"

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

func (scraper *BlueskyScraper) CanHandle(target *domain.ScrapeTarget) bool {
	return target.IsBluesky()
}

func (scraper *BlueskyScraper) Scrape(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, fetchError := scraper.webFetcher.Fetch(scrapeCtx, target.RawURL())
	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	document, parseError := BuildDocumentFromResponse(response)
	if parseError != nil {
		return nil, parseError
	}

	pageSummary := domain.NewPageSummary(target.RawURL())

	title := ExtractMeta(document, "property", "og:title")
	if title == "" {
		title = document.Find("title").Text()
	}
	pageSummary.SetTitle(title)

	description := ExtractMeta(document, "property", "og:description")
	if description == "" {
		description = ExtractMeta(document, "name", "description")
	}
	pageSummary.SetDescription(description)

	thumbnail := ExtractMeta(document, "property", "og:image")
	pageSummary.SetThumbnail(ResolveRelativeUrl(target.RawURL(), thumbnail))

	pageSummary.SetSiteName("Bluesky")
	pageSummary.SetIcon("https://bsky.app/static/favicon-32x32.png")

	videoUrl := ExtractMeta(document, "property", "og:video:url")
	if videoUrl == "" {
		videoUrl = ExtractMeta(document, "property", "og:video")
	}
	pageSummary.SetPlayer(videoUrl, 0, 0)

	pageSummary.Finalize()
	if IsContentEmpty(pageSummary) {
		return nil, nil
	}
	return pageSummary, nil
}
