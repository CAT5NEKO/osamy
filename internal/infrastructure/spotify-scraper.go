package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/user/osamy/internal/domain"
)

type SpotifyScraper struct {
	webFetcher *WebFetcher
}

const (
	spotifyEmbedBaseURL = "https://open.spotify.com/embed"
	spotifyIconURL      = "https://open.spotifycdn.com/cdn/images/favicon32.b64ecc03.png"
)

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
	kind, id, parseError := parseSpotifyContent(targetUrl)
	if parseError != nil {
		return nil, parseError
	}

	pageSummary := domain.NewPageSummary(targetUrl)
	pageSummary.SetTitle("Spotify")
	pageSummary.SetSiteName("Spotify")
	pageSummary.SetIcon(spotifyIconURL)

	playerURL := fmt.Sprintf("%s/%s/%s?utm_source=generator", spotifyEmbedBaseURL, kind, id)
	pageSummary.SetPlayer(playerURL, 0, spotifyEmbedHeight(kind))
	pageSummary.SetPlayerAllow([]string{"autoplay", "clipboard-write", "encrypted-media", "fullscreen", "picture-in-picture"})

	embedFetchURL := fmt.Sprintf("%s/%s/%s", spotifyEmbedBaseURL, kind, id)
	response, fetchError := scraper.webFetcher.Fetch(ctx, embedFetchURL)
	if fetchError == nil {
		defer response.Body.Close()
		updateSpotifySummary(pageSummary, response)
	}

	pageSummary.Finalize()
	return pageSummary, nil
}

func parseSpotifyContent(targetUrl string) (string, string, error) {
	parsedUrl, parseError := url.Parse(targetUrl)
	if parseError != nil {
		return "", "", parseError
	}

	pathParts := strings.Split(strings.Trim(parsedUrl.Path, "/"), "/")
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("unsupported spotify url format")
	}

	kind := pathParts[0]
	id := pathParts[1]
	if strings.HasPrefix(kind, "intl-") {
		if len(pathParts) < 3 {
			return "", "", fmt.Errorf("unsupported spotify url format")
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
		return "", "", fmt.Errorf("unsupported spotify content type: %s", kind)
	}

	return kind, id, nil
}

func spotifyEmbedHeight(kind string) int {
	if kind == "track" || kind == "episode" {
		return 352
	}
	return 352
}

func updateSpotifySummary(pageSummary *domain.PageSummary, response *http.Response) {
	document, err := BuildDocumentFromResponse(response)
	if err != nil {
		return
	}

	nextDataStr := document.Find("#__NEXT_DATA__").Text()
	if nextDataStr == "" {
		return
	}

	var nextData spotifyNextData
	if err := json.Unmarshal([]byte(nextDataStr), &nextData); err != nil {
		return
	}

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
