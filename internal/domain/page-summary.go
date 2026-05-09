package domain

import "strings"

type PlayerInfo struct {
	Url    string   `json:"url"`
	Width  int      `json:"width"`
	Height int      `json:"height"`
	Allow  []string `json:"allow,omitempty"`
}

type PageSummary struct {
	Title       string      `json:"title"`
	Icon        string      `json:"icon"`
	SiteName    string      `json:"siteName"`
	Sitename    string      `json:"sitename"`
	Thumbnail   string      `json:"thumbnail"`
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
	if trimmed != "" {
		summary.Icon = trimmed
	}
}

func (summary *PageSummary) SetThumbnail(thumbnail string) {
	trimmed := strings.TrimSpace(thumbnail)
	if trimmed != "" {
		summary.Thumbnail = trimmed
	}
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
	summary.Player.Url = playerUrl
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
		summary.Player = &PlayerInfo{Url: ""}
	}
	summary.ensureMediasConsistency()
}

func (summary *PageSummary) ensureMediasConsistency() {
	if summary.Medias == nil {
		summary.Medias = []string{}
	}
	if summary.Thumbnail == "" {
		return
	}
	filteredMedias := []string{}
	for _, media := range summary.Medias {
		if strings.TrimSpace(media) != "" {
			filteredMedias = append(filteredMedias, media)
		}
	}
	summary.Medias = filteredMedias

	for _, existingMedia := range summary.Medias {
		if existingMedia == summary.Thumbnail {
			return
		}
	}
	summary.Medias = append([]string{summary.Thumbnail}, summary.Medias...)
}
