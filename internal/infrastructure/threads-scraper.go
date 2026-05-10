package infrastructure

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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

func (scraper *ThreadsScraper) CanHandle(targetUrl string) bool {
	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return false
	}
	hostname := strings.ToLower(parsedUrl.Hostname())
	return hostname == "threads.net" || hostname == "www.threads.net" || hostname == "threads.com" || hostname == "www.threads.com"
}

func (scraper *ThreadsScraper) Scrape(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	fetchUrl := targetUrl
	parsed, _ := url.Parse(fetchUrl)
	if parsed != nil && strings.HasSuffix(parsed.Hostname(), "threads.com") {
		parsed.Host = strings.Replace(parsed.Host, "threads.com", "threads.net", 1)
		fetchUrl = parsed.String()
	}

	request, requestError := http.NewRequestWithContext(scrapeCtx, "GET", fetchUrl, nil)
	if requestError != nil {
		return nil, requestError
	}
	request.Header.Set("User-Agent", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)")
	request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	request.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")

	response, fetchError := scraper.webFetcher.httpClient.Do(request)
	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	finalUrl := response.Request.URL.String()

	document, parseError := BuildDocumentFromResponse(response)
	if parseError != nil {
		return nil, parseError
	}

	canonicalUrl := scraper.extractMeta(document, "property", "og:url")
	if canonicalUrl == "" {
		canonicalUrl = finalUrl
	}

	pageSummary := domain.NewPageSummary(canonicalUrl)

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
	pageSummary.SetThumbnail(ResolveRelativeUrl(canonicalUrl, thumbnail))

	pageSummary.SetSiteName("Threads")
	pageSummary.SetIcon("https://static.cdninstagram.com/rsrc.php/ye/r/lEu8iVizmNW.ico")

	videoUrl := scraper.extractMeta(document, "property", "og:video:url")
	if videoUrl == "" {
		videoUrl = scraper.extractMeta(document, "property", "og:video")
	}
	if videoUrl != "" {
		pageSummary.SetPlayer(videoUrl, 600, 338)
	}

	pageSummary.Finalize()
	return pageSummary, nil
}

func (scraper *ThreadsScraper) extractMeta(document *goquery.Document, attributeName, attributeValue string) string {
	selection := document.Find("meta[" + attributeName + "=\"" + attributeValue + "\"]")
	return selection.AttrOr("content", "")
}
