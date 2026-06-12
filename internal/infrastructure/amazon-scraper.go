package infrastructure

import (
	"context"
	"time"

	"github.com/user/osamy/internal/domain"
)

type AmazonScraper struct {
	webFetcher *WebFetcher
}

func NewAmazonScraper(webFetcher *WebFetcher) *AmazonScraper {
	return &AmazonScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *AmazonScraper) CanHandle(target *domain.ScrapeTarget) bool {
	return target.IsAmazon()
}

func (scraper *AmazonScraper) Scrape(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
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
	pageSummary.SetSiteName("Amazon")

	title := document.Find("#productTitle").Text()
	if title == "" {
		title = document.Find("meta[name='title']").AttrOr("content", "")
	}
	if title == "" {
		title = document.Find("title").Text()
	}
	pageSummary.SetTitle(title)

	image := document.Find("#landingImage").AttrOr("src", "")
	if image == "" {
		image = document.Find("meta[property='og:image']").AttrOr("content", "")
	}
	pageSummary.SetThumbnail(image)

	description := document.Find("#feature-bullets").Text()
	if description == "" {
		description = document.Find("meta[name='description']").AttrOr("content", "")
	}
	pageSummary.SetDescription(description)

	pageSummary.SetIcon("https://www.amazon.co.jp/favicon.ico")

	pageSummary.Finalize()
	return pageSummary, nil
}
