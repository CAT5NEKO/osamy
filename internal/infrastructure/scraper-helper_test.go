package infrastructure

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestEnsureAbsoluteUrl(t *testing.T) {
	cases := []struct {
		input   string
		origin  string
		expect  string
	}{
		{input: "https://example.com/a.jpg", origin: "https://origin.example", expect: "https://example.com/a.jpg"},
		{input: "//cdn.example.com/a.jpg", origin: "https://origin.example", expect: "https://cdn.example.com/a.jpg"},
		{input: "/images/a.jpg", origin: "https://origin.example", expect: "https://origin.example/images/a.jpg"},
	}

	for _, tc := range cases {
		got := EnsureAbsoluteUrl(tc.input, tc.origin)
		if got != tc.expect {
			t.Fatalf("unexpected url: got %s want %s", got, tc.expect)
		}
	}
}

func TestResolveRelativeUrl_ErrorCases(t *testing.T) {
	cases := []struct {
		name     string
		base     string
		relative string
		expect   string
	}{
		{name: "valid URLs", base: "https://example.com/page", relative: "/favicon.ico", expect: "https://example.com/favicon.ico"},
		{name: "empty relative", base: "https://example.com", relative: "", expect: ""},
		{name: "invalid base", base: "://invalid", relative: "/favicon.ico", expect: ""},
		{name: "invalid relative", base: "https://example.com", relative: "://invalid", expect: ""},
	}

	for _, tc := range cases {
		got := ResolveRelativeUrl(tc.base, tc.relative)
		if got != tc.expect {
			t.Fatalf("%s: unexpected result: got %q want %q", tc.name, got, tc.expect)
		}
	}
}

func TestResolveFileFriendlyDescription(t *testing.T) {
	cases := []struct {
		name      string
		mediaType string
		targetURL string
		expect    string
	}{
		{name: "pdf MIME", mediaType: "application/pdf", targetURL: "https://example.com/file", expect: "PDF Document"},
		{name: "pdf extension", mediaType: "", targetURL: "https://example.com/file.pdf", expect: "PDF Document"},
		{name: "excel MIME", mediaType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", targetURL: "https://example.com/file", expect: "Excel Spreadsheet"},
		{name: "word MIME", mediaType: "application/msword", targetURL: "https://example.com/file", expect: "Word Document"},
		{name: "ppt MIME", mediaType: "application/vnd.ms-powerpoint", targetURL: "https://example.com/file", expect: "PowerPoint Presentation"},
		{name: "pptx extension", mediaType: "", targetURL: "https://example.com/file.pptx", expect: "PowerPoint Presentation"},
		{name: "binary octet-stream", mediaType: "application/octet-stream", targetURL: "https://example.com/file", expect: "Binary File"},
		{name: "unknown MIME", mediaType: "application/x-some-format", targetURL: "https://example.com/file.xyz", expect: ""},
	}

	for _, tc := range cases {
		got := resolveFileFriendlyDescription(tc.mediaType, tc.targetURL)
		if got != tc.expect {
			t.Fatalf("%s: unexpected result: got %q want %q", tc.name, got, tc.expect)
		}
	}
}

func TestDetectContentKind(t *testing.T) {
	buildResponse := func(contentType string) *http.Response {
		return &http.Response{Header: http.Header{"Content-Type": {contentType}}}
	}

	cases := []struct {
		name       string
		response   *http.Response
		targetURL  string
		expectKind ContentKind
	}{
		{name: "pdf content-type", response: buildResponse("application/pdf"), targetURL: "https://example.com/file", expectKind: ContentKindPDF},
		{name: "pdf extension", response: buildResponse(""), targetURL: "https://example.com/file.pdf", expectKind: ContentKindPDF},
		{name: "word extension", response: buildResponse(""), targetURL: "https://example.com/file.docx", expectKind: ContentKindWord},
		{name: "ppt extension", response: buildResponse(""), targetURL: "https://example.com/file.pptx", expectKind: ContentKindFile},
		{name: "ppt content-type", response: buildResponse("application/vnd.openxmlformats-officedocument.presentationml.presentation"), targetURL: "https://example.com/file", expectKind: ContentKindFile},
		{name: "binary octet-stream", response: buildResponse("application/octet-stream"), targetURL: "https://example.com/file", expectKind: ContentKindFile},
		{name: "html content-type", response: buildResponse("text/html"), targetURL: "https://example.com/page", expectKind: ContentKindHTML},
	}

	for _, tc := range cases {
		got := DetectContentKind(tc.response, tc.targetURL)
		if got != tc.expectKind {
			t.Fatalf("%s: unexpected content kind: got %s want %s", tc.name, got, tc.expectKind)
		}
	}
}

func TestBuildFilePreviewSummary_Description(t *testing.T) {
	buildResponse := func(contentType string) *http.Response {
		return &http.Response{
			Header: http.Header{"Content-Type": {contentType}},
			Request: &http.Request{URL: mustParseURL("https://example.com/doc.pdf")},
		}
	}

	cases := []struct {
		name        string
		response    *http.Response
		contentKind ContentKind
		expectDesc  string
	}{
		{name: "pdf description", response: buildResponse("application/pdf"), contentKind: ContentKindPDF, expectDesc: "PDF Document"},
		{name: "excel description", response: buildResponse("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"), contentKind: ContentKindSpreadsheet, expectDesc: "Excel Spreadsheet"},
	}

	for _, tc := range cases {
		summary := BuildFilePreviewSummary("https://example.com/doc.pdf", tc.response, tc.contentKind)
		if !strings.Contains(summary.Description, tc.expectDesc) {
			t.Fatalf("%s: unexpected description: got %q want it to contain %q", tc.name, summary.Description, tc.expectDesc)
		}
	}
}

func mustParseURL(raw string) *url.URL {
	parsed, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return parsed
}
