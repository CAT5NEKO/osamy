package infrastructure

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/osamy/internal/domain"
)

type NicoNicoScraper struct {
	webFetcher *WebFetcher
}

var niconicoIdRegex = regexp.MustCompile(`watch/([a-zA-Z0-9]+)`)

func NewNicoNicoScraper(webFetcher *WebFetcher) *NicoNicoScraper {
	return &NicoNicoScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *NicoNicoScraper) CanHandle(targetUrl string) bool {
	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return false
	}
	hostname := strings.ToLower(parsedUrl.Hostname())
	return hostname == "nicovideo.jp" || hostname == "www.nicovideo.jp" || hostname == "sp.nicovideo.jp"
}

func (scraper *NicoNicoScraper) Scrape(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
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
	pageSummary.SetTitle(scraper.extractMeta(document, "property", "og:title"))
	pageSummary.SetDescription(scraper.extractMeta(document, "property", "og:description"))
	pageSummary.SetThumbnail(scraper.extractMeta(document, "property", "og:image"))
	pageSummary.SetSiteName(scraper.extractMeta(document, "property", "og:site_name"))

	icon := scraper.extractLink(document, "icon")
	if icon == "" {
		icon = scraper.extractLink(document, "shortcut icon")
	}
	pageSummary.SetIcon(ResolveRelativeUrl(targetUrl, icon))

	matches := niconicoIdRegex.FindStringSubmatch(targetUrl)
	if len(matches) > 1 {
		videoId := matches[1]
		embedUrl := "https://embed.nicovideo.jp/watch/" + videoId
		pageSummary.SetPlayer(embedUrl, 600, 338)
		pageSummary.Player.Allow = []string{"autoplay", "encrypted-media", "fullscreen", "picture-in-picture"}
	}

	pageSummary.Finalize()
	return pageSummary, nil
}

func (scraper *NicoNicoScraper) extractMeta(document *goquery.Document, attributeName, attributeValue string) string {
	selection := document.Find("meta[" + attributeName + "=\"" + attributeValue + "\"]")
	return selection.AttrOr("content", "")
}

func (scraper *NicoNicoScraper) extractLink(document *goquery.Document, relationship string) string {
	selection := document.Find("link[rel=\"" + relationship + "\"]")
	return selection.AttrOr("href", "")
}
