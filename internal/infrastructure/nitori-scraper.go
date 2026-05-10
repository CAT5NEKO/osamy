package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

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

func (scraper *NitoriScraper) CanHandle(targetUrl string) bool {
	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return false
	}
	return strings.HasSuffix(parsedUrl.Hostname(), "nitori-net.jp")
}

func (scraper *NitoriScraper) Scrape(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
	productCode := scraper.extractProductCode(targetUrl)
	if productCode != "" {
		summary, apiError := scraper.scrapeViaApi(ctx, targetUrl, productCode)
		if apiError != nil {
			return nil, apiError
		}
		return summary, nil
	}

	return scraper.scrapeViaHtml(ctx, targetUrl)
}

func (scraper *NitoriScraper) extractProductCode(targetUrl string) string {
	parsed, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return ""
	}
	pathParts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(pathParts) > 0 {
		lastPart := pathParts[len(pathParts)-1]
		if len(lastPart) > 5 {
			return lastPart
		}
	}
	return ""
}

func (scraper *NitoriScraper) scrapeViaApi(ctx context.Context, targetUrl, productCode string) (*domain.PageSummary, error) {
	apiCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
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
		title = document.Find("meta[property='og:title']").AttrOr("content", "")
	}
	if title == "" {
		title = document.Find("title").Text()
	}
	pageSummary.SetTitle(title)

	image := document.Find(".p-product-image img").First().AttrOr("src", "")
	if image == "" {
		image = document.Find("meta[property='og:image']").AttrOr("content", "")
	}
	pageSummary.SetThumbnail(image)

	pageSummary.SetIcon("https://www.nitori-net.jp/favicon.ico")

	pageSummary.Finalize()
	return pageSummary, nil
}
