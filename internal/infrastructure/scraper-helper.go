package infrastructure

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
)

var htmlTagPattern = regexp.MustCompile("<[^>]*>")

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
