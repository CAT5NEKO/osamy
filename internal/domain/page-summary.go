package domain

import (
	"net/url"
	"path"
	"strings"
)

const MaxDataUrlLength = 10 * 1024

func SanitizeUrl(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	switch parsed.Scheme {
	case "http", "https":
		return trimmed
	case "data":
		if len(trimmed) > MaxDataUrlLength {
			return ""
		}
		return trimmed
	default:
		return ""
	}
}

type PlayerInfo struct {
	Url    *string  `json:"url,omitempty"`
	Width  int      `json:"width"`
	Height int      `json:"height"`
	Allow  []string `json:"allow,omitempty"`
}

type PageSummary struct {
	Title       string      `json:"title"`
	Icon        string      `json:"icon,omitempty"`
	SiteName    string      `json:"siteName"`
	Sitename    string      `json:"sitename"`
	Thumbnail   string      `json:"thumbnail,omitempty"`
	Description string      `json:"description"`
	Url         string      `json:"url"`
	Sensitive   bool        `json:"sensitive,omitempty"`
	Medias      []string    `json:"medias"`
	Player      *PlayerInfo `json:"player,omitempty"`
}

func NewPageSummary(targetUrl string) *PageSummary {
	return &PageSummary{
		Url:    targetUrl,
		Medias: []string{},
	}
}

func (summary *PageSummary) SetTitle(title string) {
	summary.Title = strings.TrimSpace(title)
}

func (summary *PageSummary) SetDescription(description string) {
	summary.Description = strings.TrimSpace(description)
}

func (summary *PageSummary) SetIcon(icon string) {
	trimmed := strings.TrimSpace(icon)
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") && !strings.HasPrefix(trimmed, "data:") {
		return
	}
	summary.Icon = trimmed
}

func isThumbnailableUrl(rawURL string) bool {
	ext := strings.ToLower(path.Ext(rawURL))
	return ext != ".ico" && ext != ".cur"
}

func (summary *PageSummary) SetThumbnail(thumbnail string) {
	trimmed := strings.TrimSpace(thumbnail)
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		return
	}
	if !isThumbnailableUrl(trimmed) {
		return
	}
	summary.Thumbnail = trimmed
}

func (summary *PageSummary) SetSiteName(siteName string) {
	trimmed := strings.TrimSpace(siteName)
	summary.SiteName = trimmed
	if summary.Sitename == "" {
		summary.Sitename = trimmed
	}
}

func (summary *PageSummary) SetPlayer(playerUrl string, width, height int) {
	if playerUrl == "" {
		summary.Player = nil
		return
	}
	if summary.Player == nil {
		summary.Player = &PlayerInfo{}
	}
	summary.Player.Url = &playerUrl
	summary.Player.Width = width
	summary.Player.Height = height
}

func (summary *PageSummary) SetPlayerAllow(allow []string) {
	if summary.Player == nil {
		summary.Player = &PlayerInfo{}
	}
	summary.Player.Allow = allow
}

func (summary *PageSummary) Finalize() {
	if summary.Sitename == "" {
		summary.Sitename = summary.SiteName
	}
	if summary.Player == nil {
		summary.Player = &PlayerInfo{}
	}
	summary.Icon = SanitizeUrl(summary.Icon)
	summary.Thumbnail = SanitizeUrl(summary.Thumbnail)
	if summary.Player.Url != nil {
		sanitized := SanitizeUrl(*summary.Player.Url)
		if sanitized == "" {
			summary.Player.Url = nil
		} else {
			summary.Player.Url = &sanitized
		}
	}
	summary.ensureMediasConsistency()
}

func (summary *PageSummary) ensureMediasConsistency() {
	if summary.Medias == nil {
		summary.Medias = []string{}
	}
	sanitizedMedias := []string{}
	seen := map[string]bool{}
	for _, media := range summary.Medias {
		sanitized := SanitizeUrl(media)
		if sanitized == "" || seen[sanitized] {
			continue
		}
		seen[sanitized] = true
		sanitizedMedias = append(sanitizedMedias, sanitized)
	}
	summary.Medias = sanitizedMedias

	if summary.Thumbnail == "" {
		return
	}
	for _, existingMedia := range summary.Medias {
		if existingMedia == summary.Thumbnail {
			return
		}
	}
	summary.Medias = append([]string{summary.Thumbnail}, summary.Medias...)
}
