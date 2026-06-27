package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/osamy/internal/domain"
)

type TwitterScraper struct {
	webFetcher *WebFetcher
}

type fxTwitterResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Tweet   struct {
		Text   string `json:"text"`
		Author struct {
			Name       string `json:"name"`
			ScreenName string `json:"screen_name"`
		} `json:"author"`
		Media struct {
			Videos []struct {
				Url          string `json:"url"`
				ThumbnailUrl string `json:"thumbnail_url"`
				Width        int    `json:"width"`
				Height       int    `json:"height"`
			} `json:"videos"`
			Photos []struct {
				Url string `json:"url"`
			} `json:"photos"`
		} `json:"media"`
	} `json:"tweet"`
}

func NewTwitterScraper(webFetcher *WebFetcher) *TwitterScraper {
	return &TwitterScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *TwitterScraper) CanHandle(target *domain.ScrapeTarget) bool {
	return target.IsTwitter()
}

func (scraper *TwitterScraper) Scrape(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	summary, err := scraper.scrapeViaFxTwitter(scrapeCtx, target)
	if err == nil && summary != nil {
		return summary, nil
	}
	log.Printf("fxtwitter failed for %s: %v, falling back to OGP scraping", target.RawURL(), err)

	return scraper.scrapeViaOGP(scrapeCtx, target)
}

func (scraper *TwitterScraper) scrapeViaFxTwitter(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
	apiUrl := fmt.Sprintf("https://api.fxtwitter.com%s", target.Path())

	response, fetchError := scraper.webFetcher.FetchAsBot(ctx, apiUrl)
	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	var data fxTwitterResponse
	if decodeError := json.NewDecoder(response.Body).Decode(&data); decodeError != nil {
		return nil, decodeError
	}

	if data.Code != 200 {
		return nil, fmt.Errorf("fxtwitter api returned error: %s", data.Message)
	}

	pageSummary := domain.NewPageSummary(target.RawURL())

	title := fmt.Sprintf("%s (@%s)", data.Tweet.Author.Name, data.Tweet.Author.ScreenName)
	pageSummary.SetTitle(title)
	pageSummary.SetDescription(data.Tweet.Text)
	pageSummary.SetSiteName("X (formerly Twitter)")
	pageSummary.SetIcon("https://abs.twimg.com/favicons/twitter.3.ico")

	if len(data.Tweet.Media.Videos) > 0 {
		video := data.Tweet.Media.Videos[0]
		pageSummary.SetThumbnail(video.ThumbnailUrl)
	} else if len(data.Tweet.Media.Photos) > 0 {
		photo := data.Tweet.Media.Photos[0]
		pageSummary.SetThumbnail(photo.Url)
	}

	pageSummary.Finalize()
	return pageSummary, nil
}

func (scraper *TwitterScraper) scrapeViaOGP(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, fetchError := scraper.webFetcher.FetchAsFacebookBot(scrapeCtx, target.RawURL())
	if fetchError != nil {
		response, fetchError = scraper.webFetcher.FetchAsBot(scrapeCtx, target.RawURL())
		if fetchError != nil {
			return nil, fetchError
		}
	}
	defer response.Body.Close()

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
		thumbnail = extractImageFromJSONLDGeneral(document)
	}
	pageSummary.SetThumbnail(ResolveRelativeUrl(target.RawURL(), thumbnail))

	pageSummary.SetSiteName("X (formerly Twitter)")
	pageSummary.SetIcon(ResolveRelativeUrl(target.RawURL(), scraper.extractIconFromTwitter(document)))

	pageSummary.Finalize()
	if IsEmptyPreview(pageSummary) {
		return nil, nil
	}
	return pageSummary, nil
}

func (scraper *TwitterScraper) extractIconFromTwitter(document *goquery.Document) string {
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
	if icon != "" {
		return icon
	}

	return "https://abs.twimg.com/favicons/twitter.3.ico"
}

func extractImageFromJSONLDGeneral(document *goquery.Document) string {
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
