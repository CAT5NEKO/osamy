package application

import (
	"context"
	"log"

	"github.com/user/osamy/internal/domain"
)

type SummaryApplicationService struct {
	scrapers  []domain.ScraperDriver
	cache     domain.CacheRepository
	semaphore chan struct{}
}

func NewSummaryApplicationService(scrapers []domain.ScraperDriver, cache domain.CacheRepository, maxConcurrency int) *SummaryApplicationService {
	return &SummaryApplicationService{
		scrapers:  scrapers,
		cache:     cache,
		semaphore: make(chan struct{}, maxConcurrency),
	}
}

func (service *SummaryApplicationService) GetSummary(ctx context.Context, url string) (*domain.PageSummary, error) {
	cachedSummary, cacheError := service.cache.Get(ctx, url)
	if cacheError == nil && cachedSummary != nil {
		return cachedSummary, nil
	}
	if cacheError != nil {
		log.Printf("Cache access failed: %v", cacheError)
	}

	service.semaphore <- struct{}{}
	defer func() { <-service.semaphore }()

	for _, scraper := range service.scrapers {
		if scraper.CanHandle(url) {
			scrapedSummary, scrapeError := scraper.Scrape(ctx, url)
			if scrapeError != nil {
				log.Printf("Scraper failed for %s: %v", url, scrapeError)
				return nil, nil
			}
			if scrapedSummary != nil {
				_ = service.cache.Set(ctx, url, scrapedSummary)
				return scrapedSummary, nil
			}
		}
	}

	return nil, nil
}
