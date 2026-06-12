package domain

import (
	"net/url"
	"strings"
)

type ScrapeTarget struct {
	rawURL    string
	parsedURL *url.URL
}

func NewScrapeTarget(rawURL string) (*ScrapeTarget, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	return &ScrapeTarget{
		rawURL:    rawURL,
		parsedURL: parsed,
	}, nil
}

func (target *ScrapeTarget) RawURL() string {
	return target.rawURL
}

func (target *ScrapeTarget) Hostname() string {
	return strings.ToLower(target.parsedURL.Hostname())
}

func (target *ScrapeTarget) Path() string {
	return target.parsedURL.Path
}

func (target *ScrapeTarget) ReplaceHost(newHost string) string {
	copied := *target.parsedURL
	copied.Host = newHost
	return copied.String()
}

func (target *ScrapeTarget) IsInstagram() bool {
	hostname := target.Hostname()
	return hostname == "instagram.com" || hostname == "www.instagram.com"
}

func (target *ScrapeTarget) IsTikTok() bool {
	hostname := target.Hostname()
	return hostname == "tiktok.com" || hostname == "www.tiktok.com"
}

func (target *ScrapeTarget) IsPixiv() bool {
	hostname := target.Hostname()
	return hostname == "pixiv.net" || hostname == "www.pixiv.net"
}

func (target *ScrapeTarget) IsGoogleMaps() bool {
	hostname := target.Hostname()
	if hostname == "maps.app.goo.gl" {
		return true
	}
	if strings.HasPrefix(hostname, "maps.google.") {
		return true
	}
	isGoogle := hostname == "google.com" ||
		hostname == "www.google.com" ||
		strings.HasSuffix(hostname, ".google.com") ||
		strings.HasSuffix(hostname, ".google.co.jp")
	return isGoogle && strings.HasPrefix(target.Path(), "/maps")
}
