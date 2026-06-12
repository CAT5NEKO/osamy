package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

	apiUrl := fmt.Sprintf("https://api.fxtwitter.com%s", target.Path())

	response, fetchError := scraper.webFetcher.FetchAsBot(scrapeCtx, apiUrl)
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
