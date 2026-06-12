package domain

import "testing"

func TestScrapeTargetClassifications(t *testing.T) {
	tests := []struct {
		url          string
		isInstagram  bool
		isTikTok     bool
		isPixiv      bool
		isGoogleMaps bool
	}{
		{
			url:         "https://instagram.com/p/12345",
			isInstagram: true,
		},
		{
			url:         "https://www.instagram.com/reel/123",
			isInstagram: true,
		},
		{
			url:      "https://tiktok.com/@user/video/123",
			isTikTok: true,
		},
		{
			url:      "https://www.tiktok.com/t/Z123",
			isTikTok: true,
		},
		{
			url:     "https://pixiv.net/artworks/123",
			isPixiv: true,
		},
		{
			url:     "https://www.pixiv.net/member_illust.php?id=123",
			isPixiv: true,
		},
		{
			url:          "https://maps.app.goo.gl/ow6JRV2QgtYdnKGy7?g_st=ic",
			isGoogleMaps: true,
		},
		{
			url:          "https://maps.google.com/maps?q=tokyo",
			isGoogleMaps: true,
		},
		{
			url:          "https://www.google.com/maps/place/Tokyo",
			isGoogleMaps: true,
		},
		{
			url:          "https://www.google.co.jp/maps/place/Tokyo",
			isGoogleMaps: true,
		},
		{
			url:          "https://www.google.com/search?q=maps",
			isGoogleMaps: false,
		},
	}

	for _, tc := range tests {
		target, err := NewScrapeTarget(tc.url)
		if err != nil {
			t.Fatalf("failed to create scrape target for %s: %v", tc.url, err)
		}

		if target.IsInstagram() != tc.isInstagram {
			t.Errorf("unexpected IsInstagram for %s: got %t, want %t", tc.url, target.IsInstagram(), tc.isInstagram)
		}
		if target.IsTikTok() != tc.isTikTok {
			t.Errorf("unexpected IsTikTok for %s: got %t, want %t", tc.url, target.IsTikTok(), tc.isTikTok)
		}
		if target.IsPixiv() != tc.isPixiv {
			t.Errorf("unexpected IsPixiv for %s: got %t, want %t", tc.url, target.IsPixiv(), tc.isPixiv)
		}
		if target.IsGoogleMaps() != tc.isGoogleMaps {
			t.Errorf("unexpected IsGoogleMaps for %s: got %t, want %t", tc.url, target.IsGoogleMaps(), tc.isGoogleMaps)
		}
	}
}

func TestScrapeTargetReplaceHost(t *testing.T) {
	target, err := NewScrapeTarget("https://instagram.com/p/12345")
	if err != nil {
		t.Fatalf("failed to create scrape target: %v", err)
	}

	replaced := target.ReplaceHost("ddinstagram.com")
	expected := "https://ddinstagram.com/p/12345"
	if replaced != expected {
		t.Errorf("unexpected replaced URL: got %s, want %s", replaced, expected)
	}
}
