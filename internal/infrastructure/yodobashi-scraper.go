package infrastructure

import (
	"context"
	"time"

	"github.com/user/osamy/internal/domain"
)

type YodobashiScraper struct {
	webFetcher *WebFetcher
}

func NewYodobashiScraper(webFetcher *WebFetcher) *YodobashiScraper {
	return &YodobashiScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *YodobashiScraper) CanHandle(target *domain.ScrapeTarget) bool {
	return target.IsYodobashi()
}

func (scraper *YodobashiScraper) Scrape(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
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
	pageSummary.SetSiteName("ヨドバシ.com")

	title := document.Find("#productsDetails h1").Text()
	if title == "" {
		title = document.Find("title").Text()
	}
	pageSummary.SetTitle(title)

	image := document.Find("#main_img").AttrOr("src", "")
	if image == "" {
		image = document.Find("meta[property='og:image']").AttrOr("content", "")
	}
	pageSummary.SetThumbnail(image)

	pageSummary.SetIcon("https://www.yodobashi.com/favicon.ico")

	pageSummary.Finalize()
	if IsContentEmpty(pageSummary) {
		return nil, nil
	}
	return pageSummary, nil
}
