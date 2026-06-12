package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
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

func (scraper *GeneralScraper) CanHandle(targetURL string) bool {
	return true
}

func (scraper *GeneralScraper) Scrape(ctx context.Context, targetURL string) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	scrapeTarget, err := domain.NewScrapeTarget(targetURL)
	if err != nil {
		return nil, err
	}

	fetchURL, useBotUserAgent := scraper.resolveFetchParameters(scrapeTarget)

	var response *http.Response
	var fetchError error
	if useBotUserAgent {
		response, fetchError = scraper.webFetcher.FetchAsBot(scrapeCtx, fetchURL)
	} else {
		response, fetchError = scraper.webFetcher.Fetch(scrapeCtx, fetchURL)
	}

	if fetchError != nil {
		return nil, fetchError
	}
	if !useBotUserAgent && shouldRetryWithBot(response) {
		response.Body.Close()
		response, fetchError = scraper.webFetcher.FetchAsBot(scrapeCtx, fetchURL)
		if fetchError != nil {
			return nil, fetchError
		}
	}
	defer response.Body.Close()

	contentKind := DetectContentKind(response, targetURL)
	if contentKind == ContentKindPDF || contentKind == ContentKindSpreadsheet || contentKind == ContentKindWord {
		return BuildFilePreviewSummary(targetURL, response), nil
	}

	document, parseError := BuildDocumentFromResponse(response)
	if parseError != nil {
		return nil, parseError
	}

	pageSummary := domain.NewPageSummary(targetURL)
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
		thumbnail = scraper.extractMeta(document, "name", "og:image")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractMeta(document, "name", "og:image:secure_url")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractMeta(document, "name", "og:image:url")
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
	if thumbnail == "" {
		thumbnail = scraper.extractImageFromJSONLD(document)
	}
	pageSummary.SetThumbnail(ResolveRelativeUrl(targetURL, thumbnail))

	siteName := scraper.extractMeta(document, "property", "og:site_name")
	if siteName == "ddinstagram" {
		siteName = "Instagram"
	}
	pageSummary.SetSiteName(siteName)

	icon := scraper.extractIcon(document)
	pageSummary.SetIcon(ResolveRelativeUrl(targetURL, icon))

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
	value := ""
	document.Find("meta["+attributeName+"=\""+attributeValue+"\"]").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		content := strings.TrimSpace(selection.AttrOr("content", ""))
		if content == "" {
			return true
		}
		value = content
		return false
	})
	return value
}

func (scraper *GeneralScraper) extractLink(document *goquery.Document, relationship string) string {
	value := ""
	document.Find("link[rel=\""+relationship+"\"]").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		href := strings.TrimSpace(selection.AttrOr("href", ""))
		if href == "" {
			return true
		}
		value = href
		return false
	})
	return value
}

func (scraper *GeneralScraper) extractIcon(document *goquery.Document) string {
	rels := []string{"icon", "shortcut icon", "apple-touch-icon", "apple-touch-icon-precomposed", "mask-icon"}
	for _, rel := range rels {
		icon := scraper.extractLink(document, rel)
		if icon != "" {
			return icon
		}
	}

	icon := ""
	document.Find("link[rel]").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		rel := strings.ToLower(selection.AttrOr("rel", ""))
		if rel == "" || !strings.Contains(rel, "icon") {
			return true
		}
		href := strings.TrimSpace(selection.AttrOr("href", ""))
		if href == "" {
			return true
		}
		icon = href
		return false
	})
	return icon
}

func shouldRetryWithBot(response *http.Response) bool {
	if response == nil {
		return false
	}
	switch response.StatusCode {
	case http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return true
	default:
		return false
	}
}

func (scraper *GeneralScraper) extractImageFromJSONLD(document *goquery.Document) string {
	imageURL := ""
	document.Find("script[type=\"application/ld+json\"]").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		payload := strings.TrimSpace(selection.Text())
		if payload == "" {
			return true
		}
		decoder := json.NewDecoder(strings.NewReader(payload))
		decoder.UseNumber()
		var value interface{}
		if err := decoder.Decode(&value); err != nil {
			return true
		}
		imageURL = findImageInJSONLD(value)
		return imageURL == ""
	})
	return imageURL
}

func findImageInJSONLD(value interface{}) string {
	switch typed := value.(type) {
	case map[string]interface{}:
		if imageValue, ok := typed["image"]; ok {
			if url := extractImageValue(imageValue); url != "" {
				return url
			}
		}
		if imageValue, ok := typed["thumbnailUrl"]; ok {
			if url := extractImageValue(imageValue); url != "" {
				return url
			}
		}
		if imageValue, ok := typed["logo"]; ok {
			if url := extractImageValue(imageValue); url != "" {
				return url
			}
		}
		if graphValue, ok := typed["@graph"]; ok {
			if url := findImageInJSONLD(graphValue); url != "" {
				return url
			}
		}
		for _, nested := range typed {
			if url := findImageInJSONLD(nested); url != "" {
				return url
			}
		}
	case []interface{}:
		for _, item := range typed {
			if url := findImageInJSONLD(item); url != "" {
				return url
			}
		}
	case string:
		if strings.TrimSpace(typed) != "" {
			return typed
		}
	}

	return ""
}

func extractImageValue(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]interface{}:
		if url, ok := typed["url"].(string); ok {
			return url
		}
		if url, ok := typed["contentUrl"].(string); ok {
			return url
		}
		if url, ok := typed["thumbnailUrl"].(string); ok {
			return url
		}
		return findImageInJSONLD(typed)
	case []interface{}:
		for _, item := range typed {
			if url := extractImageValue(item); url != "" {
				return url
			}
		}
	}
	return ""
}

func (scraper *GeneralScraper) resolveFetchParameters(target *domain.ScrapeTarget) (string, bool) {
	if target.IsInstagram() {
		return target.ReplaceHost("ddinstagram.com"), true
	}
	if target.IsTikTok() {
		return target.ReplaceHost("vxtiktok.com"), true
	}
	if target.IsPixiv() {
		return target.ReplaceHost("phixiv.net"), true
	}
	if target.IsGoogleMaps() {
		return target.RawURL(), true
	}
	return target.RawURL(), false
}

