package infrastructure

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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

func (scraper *YodobashiScraper) CanHandle(targetUrl string) bool {
	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return false
	}
	return strings.HasSuffix(parsedUrl.Hostname(), "yodobashi.com")
}

func (scraper *YodobashiScraper) Scrape(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, fetchError := scraper.webFetcher.Fetch(scrapeCtx, targetUrl)
	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	document, parseError := goquery.NewDocumentFromReader(response.Body)
	if parseError != nil {
		return nil, parseError
	}

	pageSummary := domain.NewPageSummary(targetUrl)
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
	return pageSummary, nil
}
