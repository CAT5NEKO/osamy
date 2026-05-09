package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/osamy/internal/domain"
)

type SpotifyScraper struct {
	webFetcher *WebFetcher
}

type spotifyNextData struct {
	Props struct {
		PageProps struct {
			State struct {
				Data struct {
					Entity struct {
						Name           string `json:"name"`
						Subtitle       string `json:"subtitle"`
						VisualIdentity struct {
							Image []struct {
								Url string `json:"url"`
							} `json:"image"`
						} `json:"visualIdentity"`
					} `json:"entity"`
				} `json:"data"`
			} `json:"state"`
		} `json:"pageProps"`
	} `json:"props"`
}

func NewSpotifyScraper(webFetcher *WebFetcher) *SpotifyScraper {
	return &SpotifyScraper{
		webFetcher: webFetcher,
	}
}

func (scraper *SpotifyScraper) CanHandle(targetUrl string) bool {
	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return false
	}
	return parsedUrl.Hostname() == "open.spotify.com"
}

func (scraper *SpotifyScraper) Scrape(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return nil, parseError
	}

	pathParts := strings.Split(strings.Trim(parsedUrl.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("unsupported spotify url format")
	}

	kind := pathParts[0]
	id := pathParts[1]
	if strings.HasPrefix(kind, "intl-") {
		if len(pathParts) < 3 {
			return nil, fmt.Errorf("unsupported spotify url format")
		}
		kind = pathParts[1]
		id = pathParts[2]
	}

	validKinds := map[string]bool{
		"track":    true,
		"album":    true,
		"playlist": true,
		"show":     true,
		"episode":  true,
		"artist":   true,
	}

	if !validKinds[kind] {
		return nil, fmt.Errorf("unsupported spotify content type: %s", kind)
	}

	pageSummary := domain.NewPageSummary(targetUrl)
	pageSummary.SetTitle("Spotify")
	pageSummary.SetSiteName("Spotify")
	pageSummary.SetIcon("https://open.spotifycdn.com/cdn/images/favicon32.b64ecc03.png")

	playerURL := fmt.Sprintf("https://open.spotify.com/embed/%s/%s?utm_source=generator", kind, id)
	height := 352
	if kind == "track" || kind == "episode" {
		height = 352
	}

	pageSummary.SetPlayer(playerURL, 0, height)
	pageSummary.SetPlayerAllow([]string{"autoplay", "clipboard-write", "encrypted-media", "fullscreen", "picture-in-picture"})

	embedFetchURL := fmt.Sprintf("https://open.spotify.com/embed/%s/%s", kind, id)
	response, fetchError := scraper.webFetcher.Fetch(ctx, embedFetchURL)
	if fetchError == nil {
		defer response.Body.Close()
		document, err := goquery.NewDocumentFromReader(response.Body)
		if err == nil {
			nextDataStr := document.Find("#__NEXT_DATA__").Text()
			if nextDataStr != "" {
				var nextData spotifyNextData
				if err := json.Unmarshal([]byte(nextDataStr), &nextData); err == nil {
					entity := nextData.Props.PageProps.State.Data.Entity
					if entity.Name != "" {
						title := entity.Name
						if entity.Subtitle != "" {
							title += " - " + entity.Subtitle
						}
						pageSummary.SetTitle(title)
					}
					images := entity.VisualIdentity.Image
					if len(images) > 0 {
						pageSummary.SetThumbnail(images[len(images)-1].Url)
					}
				}
			}
		}
	}

	pageSummary.Finalize()

	return pageSummary, nil
}
