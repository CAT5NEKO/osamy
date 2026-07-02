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

const (
	PdfIconDataUrl  = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxNiIgaGVpZ2h0PSIxNiIgdmlld0JveD0iMCAwIDE2IDE2Ij48cGF0aCBmaWxsPSIjZDk1MzRmIiBkPSJNMiAxaDdsNSA1djlIMnoiLz48dGV4dCB4PSIyLjUiIHk9IjEzIiBmb250LWZhbWlseT0ic2Fucy1zZXJpZiIgZm9udC1zaXplPSI1IiBmb250LXdlaWdodD0iYm9sZCIgZmlsbD0id2hpdGUiPlBERjwvdGV4dD48L3N2Zz4="
	WordIconDataUrl = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxNiIgaGVpZ2h0PSIxNiIgdmlld0JveD0iMCAwIDE2IDE2Ij48cGF0aCBmaWxsPSIjMmI1Nzk3IiBkPSJNMiAxaDdsNSA1djlIMnoiLz48dGV4dCB4PSIyLjUiIHk9IjEzIiBmb250LWZhbWlseT0ic2Fucy1zZXJpZiIgZm9udC1zaXplPSI0IiBmb250LXdlaWdodD0iYm9sZCIgZmlsbD0id2hpdGUiPkRPQzwvdGV4dD48L3N2Zz4="
	ExcelIconDataUrl = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxNiIgaGVpZ2h0PSIxNiIgdmlld0JveD0iMCAwIDE2IDE2Ij48cGF0aCBmaWxsPSIjMjE3MzQ2IiBkPSJNMiAxaDdsNSA1djlIMnoiLz48dGV4dCB4PSIyLjUiIHk9IjEzIiBmb250LWZhbWlseT0ic2Fucy1zZXJpZiIgZm9udC1zaXplPSI0LjUiIGZvbnQtd2VpZ2h0PSJib2xkIiBmaWxsPSJ3aGl0ZSI+WExTPC90ZXh0Pjwvc3ZnPg=="
	PptIconDataUrl  = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxNiIgaGVpZ2h0PSIxNiIgdmlld0JveD0iMCAwIDE2IDE2Ij48cGF0aCBmaWxsPSIjZDI0NzI2IiBkPSJNMiAxaDdsNSA1djlIMnoiLz48dGV4dCB4PSIyLjUiIHk9IjEzIiBmb250LWZhbWlseT0ic2Fucy1zZXJpZiIgZm9udC1zaXplPSI0IiBmb250LXdlaWdodD0iYm9sZCIgZmlsbD0id2hpdGUiPlBQVDwvdGV4dD48L3N2Zz4="
	FileIconDataUrl = "data:image/svg+xml;base64,PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIxNiIgaGVpZ2h0PSIxNiIgdmlld0JveD0iMCAwIDE2IDE2Ij48cGF0aCBmaWxsPSIjNjY2IiBkPSJNMiAxaDdsNSA1djlIMnoiLz48dGV4dCB4PSIyLjUiIHk9IjEyIiBmb250LWZhbWlseT0ic2Fucy1zZXJpZiIgZm9udC1zaXplPSI2IiBmb250LXdlaWdodD0iYm9sZCIgZmlsbD0id2hpdGUiPkY8L3RleHQ+PC9zdmc+"
)

var htmlTagPattern = regexp.MustCompile("<[^>]*>")

type ContentKind string

const (
	ContentKindHTML        ContentKind = "html"
	ContentKindPDF         ContentKind = "pdf"
	ContentKindSpreadsheet ContentKind = "spreadsheet"
	ContentKindWord        ContentKind = "word"
	ContentKindFile        ContentKind = "file"
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
		return ""
	}
	parsedRelative, parseError := url.Parse(relativeUrl)
	if parseError != nil {
		return ""
	}
	return parsedBase.ResolveReference(parsedRelative).String()
}

func EnsureAbsoluteUrl(targetUrl string, defaultOrigin string) string {
	if strings.HasPrefix(targetUrl, "http") {
		return targetUrl
	}
	if strings.HasPrefix(targetUrl, "//") {
		return "https:" + targetUrl
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
	case "application/vnd.ms-powerpoint", "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return ContentKindFile
	}

	extension := extractFileExtension(targetURL, response)
	switch extension {
	case ".pdf":
		return ContentKindPDF
	case ".xls", ".xlsx":
		return ContentKindSpreadsheet
	case ".doc", ".docx":
		return ContentKindWord
	case ".ppt", ".pptx":
		return ContentKindFile
	}

	if isNonHtmlContentType(mediaType) {
		return ContentKindFile
	}

	return ContentKindHTML
}

func isNonHtmlContentType(mediaType string) bool {
	if mediaType == "" {
		return false
	}
	if strings.HasPrefix(mediaType, "text/") {
		return false
	}
	return mediaType != "text/html" && mediaType != "application/xhtml+xml" &&
		mediaType != "application/xml" && mediaType != "application/json"
}

func fileIconForKind(contentKind ContentKind) string {
	switch contentKind {
	case ContentKindPDF:
		return PdfIconDataUrl
	case ContentKindWord:
		return WordIconDataUrl
	case ContentKindSpreadsheet:
		return ExcelIconDataUrl
	case ContentKindFile:
		return PptIconDataUrl
	default:
		return FileIconDataUrl
	}
}

func BuildFilePreviewSummary(targetURL string, response *http.Response, contentKind ContentKind) *domain.PageSummary {
	pageSummary := domain.NewPageSummary(targetURL)
	pageSummary.SetTitle(resolveFileTitle(targetURL, response))
	pageSummary.SetDescription(resolveFileDescription(response, targetURL, contentKind))
	pageSummary.SetSiteName(resolveFileSiteName(targetURL, response))
	pageSummary.SetIcon(fileIconForKind(contentKind))
	pageSummary.Finalize()
	return pageSummary
}

func resolveFileIcon(targetURL string, response *http.Response) string {
	parsedURL := resolveURLForFile(targetURL, response)
	if parsedURL == nil || parsedURL.Hostname() == "" {
		return ""
	}
	return fmt.Sprintf("https://%s/favicon.ico", parsedURL.Hostname())
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

func resolveFileDescription(response *http.Response, targetURL string, contentKind ContentKind) string {
	mediaType := normalizeMediaType(response.Header.Get("Content-Type"))
	if friendly := resolveFileFriendlyDescription(mediaType, targetURL); friendly != "" {
		return friendly
	}
	if mediaType != "" {
		return mediaType
	}
	return "binary file"
}

func resolveFileFriendlyDescription(mediaType string, targetURL string) string {
	if friendly, ok := mimeTypeFriendlyNames[mediaType]; ok {
		return friendly
	}
	extension := strings.ToLower(path.Ext(targetURL))
	if friendly, ok := extensionFriendlyNames[extension]; ok {
		return friendly
	}
	return ""
}

var mimeTypeFriendlyNames = map[string]string{
	"application/pdf":                                                     "PDF Document",
	"application/msword":                                                  "Word Document",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "Word Document",
	"application/vnd.ms-excel":                                            "Excel Spreadsheet",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":   "Excel Spreadsheet",
	"application/vnd.ms-powerpoint":                                       "PowerPoint Presentation",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": "PowerPoint Presentation",
	"application/octet-stream":                                            "Binary File",
	"text/plain":                                                          "Text File",
	"text/csv":                                                            "CSV Spreadsheet",
	"application/zip":                                                     "ZIP Archive",
	"application/gzip":                                                    "GZIP Archive",
	"application/x-rar-compressed":                                        "RAR Archive",
	"application/x-7z-compressed":                                         "7z Archive",
	"application/rtf":                                                     "Rich Text Document",
}

var extensionFriendlyNames = map[string]string{
	".pdf":   "PDF Document",
	".doc":   "Word Document",
	".docx":  "Word Document",
	".xls":   "Excel Spreadsheet",
	".xlsx":  "Excel Spreadsheet",
	".ppt":   "PowerPoint Presentation",
	".pptx":  "PowerPoint Presentation",
	".txt":   "Text File",
	".csv":   "CSV Spreadsheet",
	".zip":   "ZIP Archive",
	".gz":    "GZIP Archive",
	".rar":   "RAR Archive",
	".7z":    "7z Archive",
	".rtf":   "Rich Text Document",
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

func IsContentEmpty(summary *domain.PageSummary) bool {
	if summary == nil {
		return true
	}
	return strings.TrimSpace(summary.Title) == "" &&
		strings.TrimSpace(summary.Description) == "" &&
		strings.TrimSpace(summary.Thumbnail) == ""
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
	if summary.Player != nil && summary.Player.Url != nil && strings.TrimSpace(*summary.Player.Url) != "" {
		return false
	}
	return true
}

func ExtractMeta(document *goquery.Document, attributeName, attributeValue string) string {
	value := ""
	document.Find("meta["+attributeName+"=\""+attributeValue+"\"]").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		content := strings.TrimSpace(selection.AttrOr("content", ""))
		if content == "" {
			return true
		}
		value = content
		return false
	})
	return value
}

func ExtractLink(document *goquery.Document, relationship string) string {
	value := ""
	document.Find("link[rel=\""+relationship+"\"]").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		href := strings.TrimSpace(selection.AttrOr("href", ""))
		if href == "" {
			return true
		}
		value = href
		return false
	})
	return value
}
