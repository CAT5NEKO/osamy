package application

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"

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

	scrapeTarget, err := domain.NewScrapeTarget(url)
	if err != nil {
		return nil, err
	}

	service.semaphore <- struct{}{}
	defer func() { <-service.semaphore }()

	for _, scraper := range service.scrapers {
		if scraper.CanHandle(scrapeTarget) {
			scrapedSummary, scrapeError := safeScrape(scraper, ctx, scrapeTarget)
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

func safeScrape(scraper domain.ScraperDriver, ctx context.Context, target *domain.ScrapeTarget) (summary *domain.PageSummary, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("Scraper panic for %s: %v", target.RawURL(), recovered)
			log.Printf("Scraper panic stack: %s", string(debug.Stack()))
			err = fmt.Errorf("scraper panic: %v", recovered)
			summary = nil
		}
	}()

	return scraper.Scrape(ctx, target)
}
