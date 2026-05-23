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

type GeneralScraper struct {
	webFetcher *WebFetcher
}

func NewGeneralScraper(webFetcher *WebFetcher) *GeneralScraper {
	return &GeneralScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *GeneralScraper) CanHandle(targetUrl string) bool {
	return true
}

func (scraper *GeneralScraper) Scrape(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	fetchUrl := targetUrl
	useBotUserAgent := false

	parsedUrl, err := url.Parse(fetchUrl)
	if err == nil {
		hostname := strings.ToLower(parsedUrl.Hostname())
		if hostname == "instagram.com" || hostname == "www.instagram.com" {
			parsedUrl.Host = "ddinstagram.com"
			fetchUrl = parsedUrl.String()
			useBotUserAgent = true
		} else if hostname == "tiktok.com" || hostname == "www.tiktok.com" {
			parsedUrl.Host = "vxtiktok.com"
			fetchUrl = parsedUrl.String()
			useBotUserAgent = true
		} else if hostname == "pixiv.net" || hostname == "www.pixiv.net" {
			parsedUrl.Host = "phixiv.net"
			fetchUrl = parsedUrl.String()
			useBotUserAgent = true
		}
	}

	var response *http.Response
	var fetchError error
	if useBotUserAgent {
		response, fetchError = scraper.webFetcher.FetchAsBot(scrapeCtx, fetchUrl)
	} else {
		response, fetchError = scraper.webFetcher.Fetch(scrapeCtx, fetchUrl)
	}

	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	contentKind := DetectContentKind(response, targetUrl)
	if contentKind == ContentKindPDF || contentKind == ContentKindSpreadsheet || contentKind == ContentKindWord {
		return BuildFilePreviewSummary(targetUrl, response), nil
	}

	document, parseError := BuildDocumentFromResponse(response)
	if parseError != nil {
		return nil, parseError
	}

	pageSummary := domain.NewPageSummary(targetUrl)
	title := scraper.extractMeta(document, "property", "og:title")
	if title == "" {
		title = scraper.extractMeta(document, "name", "twitter:title")
	}
	if title == "" {
		title = scraper.extractMeta(document, "property", "twitter:title")
	}
	if title == "" {
		title = document.Find("title").Text()
	}
	pageSummary.SetTitle(title)

	description := scraper.extractMeta(document, "property", "og:description")
	if description == "" {
		description = scraper.extractMeta(document, "name", "twitter:description")
	}
	if description == "" {
		description = scraper.extractMeta(document, "property", "twitter:description")
	}
	if description == "" {
		description = scraper.extractMeta(document, "name", "description")
	}
	pageSummary.SetDescription(description)

	thumbnail := scraper.extractMeta(document, "property", "og:image")
	if thumbnail == "" {
		thumbnail = scraper.extractMeta(document, "property", "og:image:secure_url")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractMeta(document, "property", "og:image:url")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractMeta(document, "name", "twitter:image")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractMeta(document, "name", "twitter:image:src")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractMeta(document, "property", "twitter:image")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractMeta(document, "itemprop", "image")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractLink(document, "image_src")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractMeta(document, "name", "thumbnail")
	}
	pageSummary.SetThumbnail(ResolveRelativeUrl(targetUrl, thumbnail))

	siteName := scraper.extractMeta(document, "property", "og:site_name")
	if siteName == "ddinstagram" {
		siteName = "Instagram"
	}
	pageSummary.SetSiteName(siteName)

	icon := scraper.extractLink(document, "icon")
	if icon == "" {
		icon = scraper.extractLink(document, "shortcut icon")
	}
	pageSummary.SetIcon(ResolveRelativeUrl(targetUrl, icon))

	videoURL := scraper.extractMeta(document, "property", "og:video:url")
	if videoURL == "" {
		videoURL = scraper.extractMeta(document, "property", "og:video")
	}

	twitterCard := scraper.extractMeta(document, "name", "twitter:card")
	if twitterCard == "" {
		twitterCard = scraper.extractMeta(document, "property", "twitter:card")
	}

	if videoURL == "" && twitterCard != "summary_large_image" {
		videoURL = scraper.extractMeta(document, "name", "twitter:player:stream")
		if videoURL == "" {
			videoURL = scraper.extractMeta(document, "property", "twitter:player:stream")
		}
		if videoURL == "" {
			videoURL = scraper.extractMeta(document, "name", "twitter:player")
		}
		if videoURL == "" {
			videoURL = scraper.extractMeta(document, "property", "twitter:player")
		}
	}
	if strings.TrimSpace(videoURL) != "" {
		pageSummary.SetPlayer(videoURL, 0, 0)
	}

	pageSummary.Finalize()
	if IsEmptyPreview(pageSummary) {
		return nil, nil
	}
	return pageSummary, nil
}

func (scraper *GeneralScraper) extractMeta(document *goquery.Document, attributeName, attributeValue string) string {
	selection := document.Find("meta[" + attributeName + "=\"" + attributeValue + "\"]")
	return selection.AttrOr("content", "")
}

func (scraper *GeneralScraper) extractLink(document *goquery.Document, relationship string) string {
	selection := document.Find("link[rel=\"" + relationship + "\"]")
	return selection.AttrOr("href", "")
}
