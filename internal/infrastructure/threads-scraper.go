package infrastructure

import (
	"context"
	"net/http"
	"time"

	"github.com/user/osamy/internal/domain"
)

type ThreadsScraper struct {
	webFetcher *WebFetcher
}

func NewThreadsScraper(webFetcher *WebFetcher) *ThreadsScraper {
	return &ThreadsScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *ThreadsScraper) CanHandle(target *domain.ScrapeTarget) bool {
	return target.IsThreads()
}

func (scraper *ThreadsScraper) Scrape(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	fetchUrl := target.ReplaceHostSuffix("threads.com", "threads.net")

	request, requestError := http.NewRequestWithContext(scrapeCtx, "GET", fetchUrl, nil)
	if requestError != nil {
		return nil, requestError
	}
	request.Header.Set("User-Agent", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	request.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")

	response, fetchError := scraper.webFetcher.Do(request)
	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	finalUrl := response.Request.URL.String()

	document, parseError := BuildDocumentFromResponse(response)
	if parseError != nil {
		return nil, parseError
	}

	canonicalUrl := ExtractMeta(document, "property", "og:url")
	if canonicalUrl == "" {
		canonicalUrl = finalUrl
	}

	pageSummary := domain.NewPageSummary(canonicalUrl)

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
	pageSummary.SetThumbnail(ResolveRelativeUrl(canonicalUrl, thumbnail))

	pageSummary.SetSiteName("Threads")
	pageSummary.SetIcon("https://static.cdninstagram.com/rsrc.php/ye/r/lEu8iVizmNW.ico")

	videoUrl := ExtractMeta(document, "property", "og:video:url")
	if videoUrl == "" {
		videoUrl = ExtractMeta(document, "property", "og:video")
	}
	if videoUrl != "" {
		pageSummary.SetPlayer(videoUrl, 600, 338)
	}

	pageSummary.Finalize()
	if IsContentEmpty(pageSummary) {
		return nil, nil
	}
	return pageSummary, nil
}
