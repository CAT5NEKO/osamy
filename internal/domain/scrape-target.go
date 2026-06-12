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

func (target *ScrapeTarget) ReplaceHostSuffix(old, newSuffix string) string {
	copied := *target.parsedURL
	if strings.HasSuffix(copied.Host, old) {
		copied.Host = strings.Replace(copied.Host, old, newSuffix, 1)
	}
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

func (target *ScrapeTarget) IsAmazon() bool {
	hostname := target.Hostname()
	return strings.HasSuffix(hostname, "amazon.co.jp") ||
		strings.HasSuffix(hostname, "amazon.com") ||
		hostname == "amzn.asia" ||
		hostname == "amzn.to"
}

func (target *ScrapeTarget) IsYouTube() bool {
	hostname := target.Hostname()
	return strings.HasSuffix(hostname, "youtube.com") || hostname == "youtu.be"
}

func (target *ScrapeTarget) IsNitori() bool {
	hostname := target.Hostname()
	return strings.HasSuffix(hostname, "nitori-net.jp")
}

func (target *ScrapeTarget) IsSpotify() bool {
	hostname := target.Hostname()
	return strings.HasSuffix(hostname, "spotify.com")
}

func (target *ScrapeTarget) IsNicoNico() bool {
	hostname := target.Hostname()
	return strings.HasSuffix(hostname, "nicovideo.jp") || hostname == "nico.ms"
}

func (target *ScrapeTarget) IsBluesky() bool {
	hostname := target.Hostname()
	return hostname == "bsky.app" || hostname == "www.bsky.app"
}

func (target *ScrapeTarget) IsThreads() bool {
	hostname := target.Hostname()
	return hostname == "threads.net" || hostname == "www.threads.net" || hostname == "threads.com" || hostname == "www.threads.com"
}

func (target *ScrapeTarget) IsTwitter() bool {
	hostname := target.Hostname()
	return hostname == "twitter.com" || hostname == "www.twitter.com" || hostname == "x.com" || hostname == "www.x.com"
}

func (target *ScrapeTarget) IsYodobashi() bool {
	hostname := target.Hostname()
	return strings.HasSuffix(hostname, "yodobashi.com")
}
