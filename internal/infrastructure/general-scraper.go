package infrastructure

import (
	"context"
	"encoding/json"
	"math/rand"
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

func (scraper *GeneralScraper) CanHandle(target *domain.ScrapeTarget) bool {
	return true
}

func (scraper *GeneralScraper) Scrape(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	fetchURL, useBotUserAgent, useFacebookBot := scraper.resolveFetchParameters(target)

	var response *http.Response
	var fetchError error
	if useFacebookBot {
		response, fetchError = scraper.webFetcher.FetchAsFacebookBot(scrapeCtx, fetchURL)
	} else if useBotUserAgent {
		response, fetchError = scraper.webFetcher.FetchAsBot(scrapeCtx, fetchURL)
	} else {
		response, fetchError = scraper.webFetcher.Fetch(scrapeCtx, fetchURL)
	}

	if fetchError != nil {
		return nil, fetchError
	}

	if !useBotUserAgent && !useFacebookBot && shouldRetryWithBot(response) {
		response.Body.Close()

		delay := time.Duration(2000+rand.Int63n(1000)) * time.Millisecond
		select {
		case <-time.After(delay):
		case <-scrapeCtx.Done():
			return nil, scrapeCtx.Err()
		}

		response, fetchError = scraper.webFetcher.FetchAsBot(scrapeCtx, fetchURL)
		if fetchError != nil {
			return nil, fetchError
		}
	}
	defer response.Body.Close()

	contentKind := DetectContentKind(response, target.RawURL())
	if contentKind == ContentKindPDF || contentKind == ContentKindSpreadsheet || contentKind == ContentKindWord || contentKind == ContentKindFile {
		return BuildFilePreviewSummary(target.RawURL(), response, contentKind), nil
	}

	document, parseError := BuildDocumentFromResponse(response)
	if parseError != nil {
		return nil, parseError
	}

	pageSummary := domain.NewPageSummary(target.RawURL())
	title := ExtractMeta(document, "property", "og:title")
	if title == "" {
		title = ExtractMeta(document, "name", "twitter:title")
	}
	if title == "" {
		title = ExtractMeta(document, "property", "twitter:title")
	}
	if title == "" {
		title = document.Find("title").Text()
	}
	pageSummary.SetTitle(title)

	description := ExtractMeta(document, "property", "og:description")
	if description == "" {
		description = ExtractMeta(document, "name", "twitter:description")
	}
	if description == "" {
		description = ExtractMeta(document, "property", "twitter:description")
	}
	if description == "" {
		description = ExtractMeta(document, "name", "description")
	}
	pageSummary.SetDescription(description)

	thumbnail := ExtractMeta(document, "property", "og:image")
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "property", "og:image:secure_url")
	}
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "property", "og:image:url")
	}
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "name", "og:image")
	}
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "name", "og:image:secure_url")
	}
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "name", "og:image:url")
	}
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "name", "twitter:image")
	}
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "name", "twitter:image:src")
	}
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "property", "twitter:image")
	}
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "itemprop", "image")
	}
	if thumbnail == "" {
		thumbnail = ExtractLink(document, "image_src")
	}
	if thumbnail == "" {
		thumbnail = ExtractMeta(document, "name", "thumbnail")
	}
	if thumbnail == "" {
		thumbnail = scraper.extractImageFromJSONLD(document)
	}
	pageSummary.SetThumbnail(ResolveRelativeUrl(target.RawURL(), thumbnail))

	siteName := ExtractMeta(document, "property", "og:site_name")
	if siteName == "ddinstagram" {
		siteName = "Instagram"
	}
	pageSummary.SetSiteName(siteName)

	icon := scraper.extractIcon(document)
	resolvedIcon := ResolveRelativeUrl(target.RawURL(), icon)
	pageSummary.SetIcon(resolvedIcon)
	if pageSummary.Thumbnail == "" && pageSummary.Icon != "" {
		pageSummary.SetThumbnail(pageSummary.Icon)
	}

	videoURL := ExtractMeta(document, "property", "og:video:url")
	if videoURL == "" {
		videoURL = ExtractMeta(document, "property", "og:video")
	}

	twitterCard := ExtractMeta(document, "name", "twitter:card")
	if twitterCard == "" {
		twitterCard = ExtractMeta(document, "property", "twitter:card")
	}

	if videoURL == "" && twitterCard != "summary_large_image" {
		videoURL = ExtractMeta(document, "name", "twitter:player:stream")
		if videoURL == "" {
			videoURL = ExtractMeta(document, "property", "twitter:player:stream")
		}
		if videoURL == "" {
			videoURL = ExtractMeta(document, "name", "twitter:player")
		}
		if videoURL == "" {
			videoURL = ExtractMeta(document, "property", "twitter:player")
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

func (scraper *GeneralScraper) extractIcon(document *goquery.Document) string {
	rels := []string{"icon", "shortcut icon", "apple-touch-icon", "apple-touch-icon-precomposed", "mask-icon"}
	for _, rel := range rels {
		icon := ExtractLink(document, rel)
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
	return response.StatusCode == http.StatusServiceUnavailable
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

func (scraper *GeneralScraper) resolveFetchParameters(target *domain.ScrapeTarget) (string, bool, bool) {
	if target.IsInstagram() {
		return target.RawURL(), false, true
	}
	if target.IsTikTok() {
		return target.ReplaceHost("vxtiktok.com"), true, false
	}
	if target.IsPixiv() {
		return target.ReplaceHost("phixiv.net"), true, false
	}
	if target.IsGoogleMaps() {
		return target.RawURL(), true, false
	}
	return target.RawURL(), false, false
}
