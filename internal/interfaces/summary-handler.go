package interfaces

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/user/osamy/internal/application"
	"github.com/user/osamy/internal/domain"
)

type SummaryHandler struct {
	summaryService *application.SummaryApplicationService
}

func NewSummaryHandler(summaryService *application.SummaryApplicationService) *SummaryHandler {
	return &SummaryHandler{
		summaryService: summaryService,
	}
}

func (handler *SummaryHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.setSecurityHeaders(writer)

	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targetUrl := request.URL.Query().Get("url")
	if targetUrl == "" {
		http.Error(writer, "url parameter is required", http.StatusBadRequest)
		return
	}

	if validationError := domain.ValidateTargetUrl(targetUrl); validationError != nil {
		log.Printf("URL validation failed for %s: %v", targetUrl, validationError)
		http.Error(writer, "invalid url", http.StatusBadRequest)
		return
	}

	pageSummary, summaryError := handler.summaryService.GetSummary(request.Context(), targetUrl)
	if summaryError != nil {
		log.Printf("Summary fetch failed for %s: %v", targetUrl, summaryError)
		http.Error(writer, "failed to process request", http.StatusInternalServerError)
		return
	}

	if pageSummary == nil {
		http.Error(writer, "failed to fetch summary", http.StatusNotFound)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	if encodeError := json.NewEncoder(writer).Encode(pageSummary); encodeError != nil {
		log.Printf("JSON encode failed: %v", encodeError)
	}
}

func (handler *SummaryHandler) setSecurityHeaders(writer http.ResponseWriter) {
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.Header().Set("X-Frame-Options", "DENY")
	writer.Header().Set("Content-Security-Policy", "default-src 'none'")
	writer.Header().Set("Cache-Control", "public, max-age=3600")
}
