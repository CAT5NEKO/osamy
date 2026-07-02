package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFinalize_PlayerPresentByDefault(t *testing.T) {
	summary := NewPageSummary("https://example.com/page")
	summary.Finalize()
	if summary.Player == nil {
		t.Fatalf("expected non-nil Player after Finalize, got nil")
	}
	if summary.Player.Url != nil {
		t.Fatalf("expected nil Player.Url by default, got %v", summary.Player.Url)
	}
}

func TestFinalize_PlayerPreservedWhenSet(t *testing.T) {
	summary := NewPageSummary("https://example.com/page")
	summary.SetPlayer("https://player.example.com/video", 600, 338)
	summary.Finalize()
	if summary.Player == nil {
		t.Fatalf("expected non-nil Player after Finalize when set")
	}
	if summary.Player.Url == nil || *summary.Player.Url != "https://player.example.com/video" {
		t.Fatalf("unexpected Player.Url: got %v", summary.Player.Url)
	}
	if summary.Player.Width != 600 || summary.Player.Height != 338 {
		t.Fatalf("unexpected Player dimensions: got %dx%d", summary.Player.Width, summary.Player.Height)
	}
}

func TestFinalize_PlayerAllowPreservedWhenSet(t *testing.T) {
	summary := NewPageSummary("https://example.com")
	summary.SetPlayer("https://player.example.com/video", 600, 338)
	summary.SetPlayerAllow([]string{"autoplay", "fullscreen"})
	summary.Finalize()
	if summary.Player == nil {
		t.Fatalf("expected non-nil Player")
	}
	if len(summary.Player.Allow) != 2 {
		t.Fatalf("expected 2 allow values, got %d", len(summary.Player.Allow))
	}
}

func TestSetIcon_ValidatesUrl(t *testing.T) {
	cases := []struct {
		input  string
		expect string
	}{
		{input: "https://example.com/icon.png", expect: "https://example.com/icon.png"},
		{input: "http://example.com/icon.png", expect: "http://example.com/icon.png"},
		{input: "", expect: ""},
		{input: "  ", expect: ""},
		{input: "/favicon.ico", expect: ""},
		{input: "//cdn.example.com/icon.ico", expect: ""},
	}

	for _, tc := range cases {
		summary := NewPageSummary("https://example.com")
		summary.SetIcon(tc.input)
		if summary.Icon != tc.expect {
			t.Fatalf("SetIcon(%q): got %q want %q", tc.input, summary.Icon, tc.expect)
		}
	}
}

func TestSetThumbnail_ValidatesUrl(t *testing.T) {
	summary := NewPageSummary("https://example.com")
	summary.SetThumbnail("/relative/path.jpg")
	if summary.Thumbnail != "" {
		t.Fatalf("expected empty thumbnail for relative path, got %q", summary.Thumbnail)
	}
	summary.SetThumbnail("https://cdn.example.com/image.jpg")
	if !strings.HasPrefix(summary.Thumbnail, "https://") {
		t.Fatalf("expected https thumbnail, got %q", summary.Thumbnail)
	}
}

func TestSetThumbnail_RejectsIco(t *testing.T) {
	cases := []struct {
		url    string
		accept bool
	}{
		{"https://example.com/favicon.ico", false},
		{"https://example.com/favicon.cur", false},
		{"https://example.com/image.ICO", false},
		{"https://example.com/image.png", true},
		{"https://example.com/image.jpg?ico", true},
		{"https://example.com/image", true},
	}
	for _, tc := range cases {
		summary := NewPageSummary("https://example.com")
		summary.SetThumbnail(tc.url)
		if tc.accept && summary.Thumbnail == "" {
			t.Fatalf("expected thumbnail to be accepted: %q", tc.url)
		}
		if !tc.accept && summary.Thumbnail != "" {
			t.Fatalf("expected thumbnail to be rejected: %q", tc.url)
		}
	}
}

func TestSanitizeUrl(t *testing.T) {
	cases := []struct {
		input  string
		expect string
	}{
		{"", ""},
		{"  ", ""},
		{"not-a-url", ""},
		{"javascript:alert(1)", ""},
		{"file:///etc/passwd", ""},
		{"https://example.com/image.jpg", "https://example.com/image.jpg"},
		{"http://example.com/image.jpg", "http://example.com/image.jpg"},
		{"data:image/svg+xml;base64,dGVzdA==", "data:image/svg+xml;base64,dGVzdA=="},
	}
	for _, tc := range cases {
		got := SanitizeUrl(tc.input)
		if got != tc.expect {
			t.Fatalf("SanitizeUrl(%q): got %q want %q", tc.input, got, tc.expect)
		}
	}
}

func TestSanitizeUrl_RejectsLargeDataUri(t *testing.T) {
	large := strings.Repeat("a", MaxDataUrlLength+1)
	input := "data:text/plain," + large
	got := SanitizeUrl(input)
	if got != "" {
		t.Fatalf("expected empty for large data URI (len=%d)", len(input))
	}
}

func TestFinalize_SanitizesIcon(t *testing.T) {
	summary := NewPageSummary("https://example.com")
	summary.Icon = "//evil.com/icon.png"
	summary.Finalize()
	if summary.Icon != "" {
		t.Fatalf("expected empty icon after Finalize, got %q", summary.Icon)
	}
}

func TestFinalize_SanitizesThumbnail(t *testing.T) {
	summary := NewPageSummary("https://example.com")
	summary.Thumbnail = "//evil.com/img.png"
	summary.Finalize()
	if summary.Thumbnail != "" {
		t.Fatalf("expected empty thumbnail after Finalize, got %q", summary.Thumbnail)
	}
}

func TestFinalize_SanitizesPlayerUrl(t *testing.T) {
	summary := NewPageSummary("https://example.com")
	summary.SetPlayer("//evil.com/player", 640, 360)
	summary.Finalize()
	if summary.Player.Url != nil {
		t.Fatalf("expected nil Player.Url after Finalize, got %v", summary.Player.Url)
	}
}

func TestFinalize_SanitizesMedias(t *testing.T) {
	summary := NewPageSummary("https://example.com")
	summary.Medias = []string{"//evil.com/img.png", "", "https://good.com/img.jpg", "https://good.com/img.jpg"}
	summary.Finalize()
	if len(summary.Medias) != 1 {
		t.Fatalf("expected 1 media, got %d: %v", len(summary.Medias), summary.Medias)
	}
	if summary.Medias[0] != "https://good.com/img.jpg" {
		t.Fatalf("unexpected media: %q", summary.Medias[0])
	}
}

func TestSetIcon_AcceptsDataUri(t *testing.T) {
	summary := NewPageSummary("https://example.com")
	summary.SetIcon("data:image/svg+xml;base64,dGVzdA==")
	if summary.Icon != "data:image/svg+xml;base64,dGVzdA==" {
		t.Fatalf("expected data URI icon, got %q", summary.Icon)
	}
}

func TestFinalize_ProducesValidJson(t *testing.T) {
	summary := NewPageSummary("https://example.com")
	summary.SetIcon("https://example.com/icon.png")
	summary.SetThumbnail("https://example.com/thumb.png")
	summary.SetSiteName("Example")
	summary.SetTitle("Test")
	summary.SetDescription("A test")
	summary.Finalize()
	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if !strings.Contains(string(data), "thumb.png") {
		t.Fatalf("expected thumb.png in JSON, got %s", data)
	}
	if !strings.Contains(string(data), "icon.png") {
		t.Fatalf("expected icon.png in JSON, got %s", data)
	}
}
