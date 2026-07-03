package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/osamy/internal/domain"
)

type NitoriScraper struct {
	webFetcher *WebFetcher
}

type nitoriApiResponse struct {
	SkuData struct {
		Name       string `json:"name"`
		CatchCopy  string `json:"catchCopy"`
		MediasList []struct {
			URL string `json:"url"`
		} `json:"mediasList"`
	} `json:"skuData"`
	Price struct {
		Value float64 `json:"value"`
	} `json:"price"`
}

func NewNitoriScraper(webFetcher *WebFetcher) *NitoriScraper {
	return &NitoriScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *NitoriScraper) CanHandle(target *domain.ScrapeTarget) bool {
	return target.IsNitori()
}

func (scraper *NitoriScraper) Scrape(ctx context.Context, target *domain.ScrapeTarget) (*domain.PageSummary, error) {
	productCode := extractNitoriProductCode(target.Path())
	if productCode != "" {
		summary, apiError := scraper.scrapeViaApi(ctx, target.RawURL(), productCode)
		if apiError == nil && summary != nil && summary.Thumbnail != "" {
			return summary, nil
		}
		log.Printf("Nitori API failed for %s: %v, falling back to HTML", target.RawURL(), apiError)
	}

	return scraper.scrapeViaHtml(ctx, target.RawURL())
}

func extractNitoriProductCode(urlPath string) string {
	pathParts := strings.Split(strings.Trim(urlPath, "/"), "/")
	if len(pathParts) > 0 {
		lastPart := pathParts[len(pathParts)-1]
		if len(lastPart) >= 8 {
			return lastPart
		}
	}
	return ""
}

func (scraper *NitoriScraper) scrapeViaApi(ctx context.Context, targetUrl, productCode string) (*domain.PageSummary, error) {
	apiCtx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	apiUrl := fmt.Sprintf("https://www.nitori-net.jp/occ/v2/nitorinet/nitori/products/%s?lang=ja&curr=JPY", productCode)
	request, requestError := http.NewRequestWithContext(apiCtx, "GET", apiUrl, nil)
	if requestError != nil {
		return nil, requestError
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	request.Header.Set("Accept", "application/json, text/plain, */*")
	request.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")
	request.Header.Set("Referer", targetUrl)
	request.Header.Set("Origin", "https://www.nitori-net.jp")
	request.Header.Set("Sec-Fetch-Dest", "empty")
	request.Header.Set("Sec-Fetch-Mode", "cors")
	request.Header.Set("Sec-Fetch-Site", "same-origin")

	response, fetchError := scraper.webFetcher.Do(request)
	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusTooManyRequests || response.StatusCode == http.StatusServiceUnavailable {
		time.Sleep(500 * time.Millisecond)
		request2, _ := http.NewRequestWithContext(apiCtx, "GET", apiUrl, nil)
		request2.Header = request.Header.Clone()
		response2, retryError := scraper.webFetcher.Do(request2)
		if retryError != nil {
			return nil, retryError
		}
		response = response2
		defer response.Body.Close()
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned status %d", response.StatusCode)
	}

	var apiResponse nitoriApiResponse
	if decodeError := json.NewDecoder(response.Body).Decode(&apiResponse); decodeError != nil {
		return nil, decodeError
	}

	if apiResponse.SkuData.Name == "" {
		return nil, fmt.Errorf("api returned empty product data")
	}

	summary := domain.NewPageSummary(targetUrl)
	summary.SetTitle(apiResponse.SkuData.Name)
	summary.SetSiteName("ニトリネット")
	summary.SetIcon("https://www.nitori-net.jp/favicon.ico")

	description := StripHtmlTags(apiResponse.SkuData.CatchCopy)
	if apiResponse.Price.Value > 0 {
		if description != "" {
			description += " | "
		}
		description += fmt.Sprintf("価格: ¥%s", FormatPriceWithComma(apiResponse.Price.Value))
	}
	summary.SetDescription(description)

	nitoriOrigin := "https://www.nitori-net.jp"
	if len(apiResponse.SkuData.MediasList) > 0 {
		summary.SetThumbnail(EnsureAbsoluteUrl(apiResponse.SkuData.MediasList[0].URL, nitoriOrigin))
		for _, media := range apiResponse.SkuData.MediasList {
			summary.Medias = append(summary.Medias, EnsureAbsoluteUrl(media.URL, nitoriOrigin))
		}
	}

	summary.Finalize()
	if IsContentEmpty(summary) {
		return nil, nil
	}
	return summary, nil
}

func (scraper *NitoriScraper) scrapeViaHtml(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
	htmlCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, fetchError := scraper.webFetcher.Fetch(htmlCtx, targetUrl)
	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	document, parseError := BuildDocumentFromResponse(response)
	if parseError != nil {
		return nil, parseError
	}

	pageSummary := domain.NewPageSummary(targetUrl)
	pageSummary.SetSiteName("ニトリネット")

	title := document.Find(".p-product-name").First().Text()
	if title == "" {
		title = ExtractMeta(document, "property", "og:title")
	}
	if title == "" {
		title = document.Find("title").Text()
	}
	pageSummary.SetTitle(title)

	image := resolveNitoriImage(document, targetUrl)
	pageSummary.SetThumbnail(image)

	pageSummary.SetIcon(ResolveRelativeUrl(targetUrl, scraper.extractNitoriIcon(document)))

	pageSummary.Finalize()
	if IsContentEmpty(pageSummary) {
		return nil, nil
	}
	return pageSummary, nil
}

func firstNonEmpty(extractors ...func() string) string {
	for _, ext := range extractors {
		if s := ext(); s != "" {
			return s
		}
	}
	return ""
}

func resolveNitoriImage(document *goquery.Document, baseUrl string) string {
	image := firstNonEmpty(
		func() string { return ExtractMeta(document, "property", "og:image") },
		func() string { return ExtractMeta(document, "property", "og:image:secure_url") },
		func() string { return ExtractMeta(document, "property", "og:image:url") },
		func() string { return ExtractMeta(document, "name", "og:image") },
		func() string { return ExtractMeta(document, "name", "twitter:image") },
		func() string { return ExtractMeta(document, "name", "twitter:image:src") },
		func() string { return ExtractMeta(document, "itemprop", "image") },
		func() string { return ExtractLink(document, "image_src") },
		func() string { return document.Find(".p-product-image img").First().AttrOr("src", "") },
		func() string { return document.Find(".ph-item-img--main img").First().AttrOr("src", "") },
		func() string { return document.Find(".p-introduction-block img").First().AttrOr("src", "") },
		func() string { return document.Find(".p-hero img").First().AttrOr("src", "") },
		func() string { return document.Find(".p-product-image-block img").First().AttrOr("src", "") },
		func() string { return document.Find(".p-product-slider img").First().AttrOr("src", "") },
		func() string { return document.Find(".itemImg img").First().AttrOr("src", "") },
		func() string { return document.Find(".c-product-gallery img").First().AttrOr("src", "") },
	)
	return ResolveRelativeUrl(baseUrl, image)
}

func (scraper *NitoriScraper) extractNitoriIcon(document *goquery.Document) string {
	rels := []string{"icon", "shortcut icon", "apple-touch-icon", "apple-touch-icon-precomposed"}
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
