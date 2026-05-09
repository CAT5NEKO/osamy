package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/osamy/internal/domain"
)

type YouTubeScraper struct {
	webFetcher *WebFetcher
}

type oEmbedResponse struct {
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	ProviderName string `json:"provider_name"`
	ThumbnailUrl string `json:"thumbnail_url"`
	Html         string `json:"html"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

func NewYouTubeScraper(webFetcher *WebFetcher) *YouTubeScraper {
	return &YouTubeScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *YouTubeScraper) CanHandle(targetUrl string) bool {
	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return false
	}
	hostname := parsedUrl.Hostname()
	if strings.HasSuffix(hostname, "youtube.com") || hostname == "youtu.be" {
		return true
	}
	return false
}

func (scraper *YouTubeScraper) Scrape(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
	scrapeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	oEmbedUrl := fmt.Sprintf("https://www.youtube.com/oembed?url=%s&format=json&maxwidth=600", url.QueryEscape(targetUrl))
	response, fetchError := scraper.webFetcher.Fetch(scrapeCtx, oEmbedUrl)
	if fetchError != nil {
		return nil, fetchError
	}
	defer response.Body.Close()

	var data oEmbedResponse
	if decodeError := json.NewDecoder(response.Body).Decode(&data); decodeError != nil {
		return nil, decodeError
	}

	pageSummary := domain.NewPageSummary(targetUrl)
	pageSummary.SetTitle(data.Title)
	pageSummary.SetThumbnail(data.ThumbnailUrl)
	pageSummary.SetSiteName(data.ProviderName)
	pageSummary.SetIcon("https://www.youtube.com/s/desktop/014dbbed/img/favicon_32x32.png")

	document, parseError := goquery.NewDocumentFromReader(strings.NewReader(data.Html))
	if parseError == nil {
		playerUrl := document.Find("iframe").AttrOr("src", "")
		if playerUrl != "" {
			if strings.HasPrefix(playerUrl, "//") {
				playerUrl = "https:" + playerUrl
			}
			width := data.Width
			height := data.Height
			if width < 480 {
				width = 600
				height = 338
			}
			pageSummary.SetPlayer(playerUrl, width, height)
		}
	} else {
		log.Printf("Failed to parse oEmbed HTML: %v", parseError)
	}

	pageSummary.Finalize()
	return pageSummary, nil
}
