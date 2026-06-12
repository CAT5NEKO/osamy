package infrastructure

import (
	"context"
	"regexp"
	"time"

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

func (scraper *NicoNicoScraper) CanHandle(target *domain.ScrapeTarget) bool {
	return target.IsNicoNico()
}

func (scraper *NicoNicoScraper) Scrape(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
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
	pageSummary.SetTitle(ExtractMeta(document, "property", "og:title"))
	pageSummary.SetDescription(ExtractMeta(document, "property", "og:description"))
	pageSummary.SetThumbnail(ExtractMeta(document, "property", "og:image"))
	pageSummary.SetSiteName(ExtractMeta(document, "property", "og:site_name"))

	icon := ExtractLink(document, "icon")
	if icon == "" {
		icon = ExtractLink(document, "shortcut icon")
	}
	pageSummary.SetIcon(ResolveRelativeUrl(target.RawURL(), icon))

	matches := niconicoIdRegex.FindStringSubmatch(target.RawURL())
	if len(matches) > 1 {
		videoId := matches[1]
		embedUrl := "https://embed.nicovideo.jp/watch/" + videoId
		pageSummary.SetPlayer(embedUrl, 600, 338)
		pageSummary.Player.Allow = []string{"autoplay", "encrypted-media", "fullscreen", "picture-in-picture"}
	}

	pageSummary.Finalize()
	return pageSummary, nil
}
