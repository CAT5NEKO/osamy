package infrastructure

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/user/osamy/internal/domain"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

var htmlTagPattern = regexp.MustCompile("<[^>]*>")

type ContentKind string

const (
	ContentKindHTML        ContentKind = "html"
	ContentKindPDF         ContentKind = "pdf"
	ContentKindSpreadsheet ContentKind = "spreadsheet"
	ContentKindWord        ContentKind = "word"
)

func StripHtmlTags(input string) string {
	return html.UnescapeString(htmlTagPattern.ReplaceAllString(input, " "))
}

func FormatPriceWithComma(price float64) string {
	integerPart := fmt.Sprintf("%.0f", price)
	if len(integerPart) <= 3 {
		return integerPart
	}
	var segments []string
	for i := len(integerPart); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		segments = append([]string{integerPart[start:i]}, segments...)
	}
	return strings.Join(segments, ",")
}

func ResolveRelativeUrl(baseUrl string, relativeUrl string) string {
	if relativeUrl == "" {
		return ""
	}
	parsedBase, parseError := url.Parse(baseUrl)
	if parseError != nil {
		return relativeUrl
	}
	parsedRelative, parseError := url.Parse(relativeUrl)
	if parseError != nil {
		return relativeUrl
	}
	return parsedBase.ResolveReference(parsedRelative).String()
}

func EnsureAbsoluteUrl(targetUrl string, defaultOrigin string) string {
	if strings.HasPrefix(targetUrl, "http") {
		return targetUrl
	}
	return defaultOrigin + targetUrl
}

func BuildDocumentFromResponse(response *http.Response) (*goquery.Document, error) {
	limitedReader := io.LimitReader(response.Body, MaxFetchResponseBodySize)
	previewBytes, _ := io.ReadAll(io.LimitReader(limitedReader, 8192))
	encoding, _, _ := charset.DetermineEncoding(previewBytes, response.Header.Get("Content-Type"))
	decodedReader := transform.NewReader(io.MultiReader(bytes.NewReader(previewBytes), limitedReader), encoding.NewDecoder())
	return goquery.NewDocumentFromReader(decodedReader)
}

func DetectContentKind(response *http.Response, targetURL string) ContentKind {
	mediaType := normalizeMediaType(response.Header.Get("Content-Type"))
	switch mediaType {
	case "application/pdf":
		return ContentKindPDF
	case "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return ContentKindSpreadsheet
	case "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		return ContentKindWord
	}

	extension := extractFileExtension(targetURL, response)
	switch extension {
	case ".pdf":
		return ContentKindPDF
	case ".xls", ".xlsx":
		return ContentKindSpreadsheet
	case ".doc", ".docx":
		return ContentKindWord
	default:
		return ContentKindHTML
	}
}

func BuildFilePreviewSummary(targetURL string, response *http.Response) *domain.PageSummary {
	pageSummary := domain.NewPageSummary(targetURL)
	pageSummary.SetTitle(resolveFileTitle(targetURL, response))
	pageSummary.SetDescription(resolveFileDescription(response))
	pageSummary.SetSiteName(resolveFileSiteName(targetURL, response))
	pageSummary.Finalize()
	return pageSummary
}

func resolveFileTitle(targetURL string, response *http.Response) string {
	fileName := extractFilenameFromContentDisposition(response.Header.Get("Content-Disposition"))
	if fileName != "" {
		return fileName
	}

	parsedURL := resolveURLForFile(targetURL, response)
	if parsedURL != nil {
		baseName := path.Base(parsedURL.Path)
		if baseName != "." && baseName != "/" && baseName != "" {
			return baseName
		}
	}

	if parsedURL != nil && parsedURL.Hostname() != "" {
		return parsedURL.Hostname()
	}

	return targetURL
}

func resolveFileDescription(response *http.Response) string {
	mediaType := normalizeMediaType(response.Header.Get("Content-Type"))
	if mediaType == "" {
		return "binary file"
	}
	return mediaType
}

func resolveFileSiteName(targetURL string, response *http.Response) string {
	parsedURL := resolveURLForFile(targetURL, response)
	if parsedURL == nil {
		return ""
	}
	return parsedURL.Hostname()
}

func resolveURLForFile(targetURL string, response *http.Response) *url.URL {
	if response != nil && response.Request != nil && response.Request.URL != nil {
		return response.Request.URL
	}
	parsedURL, parseError := url.Parse(targetURL)
	if parseError != nil {
		return nil
	}
	return parsedURL
}

func normalizeMediaType(contentType string) string {
	if contentType == "" {
		return ""
	}
	mediaType, _, parseError := mime.ParseMediaType(contentType)
	if parseError != nil {
		return strings.ToLower(strings.TrimSpace(contentType))
	}
	return strings.ToLower(mediaType)
}

func extractFileExtension(targetURL string, response *http.Response) string {
	parsedURL := resolveURLForFile(targetURL, response)
	if parsedURL == nil {
		return ""
	}
	return strings.ToLower(path.Ext(parsedURL.Path))
}

func extractFilenameFromContentDisposition(contentDisposition string) string {
	if contentDisposition == "" {
		return ""
	}

	_, params, parseError := mime.ParseMediaType(contentDisposition)
	if parseError != nil {
		return ""
	}

	if encodedFileName, ok := params["filename*"]; ok {
		parts := strings.SplitN(encodedFileName, "''", 2)
		if len(parts) == 2 {
			if decoded, err := url.QueryUnescape(parts[1]); err == nil {
				return decoded
			}
			return parts[1]
		}
		return encodedFileName
	}

	if fileName, ok := params["filename"]; ok {
		return fileName
	}

	return ""
}

func IsEmptyPreview(summary *domain.PageSummary) bool {
	if summary == nil {
		return true
	}
	if strings.TrimSpace(summary.Title) != "" {
		return false
	}
	if strings.TrimSpace(summary.Description) != "" {
		return false
	}
	if strings.TrimSpace(summary.Thumbnail) != "" {
		return false
	}
	if strings.TrimSpace(summary.Icon) != "" {
		return false
	}
	if strings.TrimSpace(summary.SiteName) != "" {
		return false
	}
	if strings.TrimSpace(summary.Sitename) != "" {
		return false
	}
	if len(summary.Medias) > 0 {
		for _, media := range summary.Medias {
			if strings.TrimSpace(media) != "" {
				return false
			}
		}
	}
	if summary.Player != nil && strings.TrimSpace(summary.Player.Url) != "" {
		return false
	}
	return true
}
