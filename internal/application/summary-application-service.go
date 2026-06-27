package application

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"

	"github.com/user/osamy/internal/domain"
)

type inflightEntry struct {
	done   chan struct{}
	result *domain.PageSummary
}

type SummaryApplicationService struct {
	scrapers  []domain.ScraperDriver
	cache     domain.CacheRepository
	semaphore chan struct{}
	inflight  sync.Map
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

	entry := &inflightEntry{done: make(chan struct{})}
	existing, loaded := service.inflight.LoadOrStore(url, entry)
	if loaded {
		existingEntry := existing.(*inflightEntry)
		select {
		case <-existingEntry.done:
			return existingEntry.result, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	defer service.inflight.Delete(url)

	select {
	case service.semaphore <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-service.semaphore }()

	result := service.fetchURL(url)

	entry.result = result
	close(entry.done)

	if result != nil {
		_ = service.cache.Set(context.Background(), url, result)
	}
	return result, nil
}

func (service *SummaryApplicationService) fetchURL(url string) *domain.PageSummary {
	scrapeTarget, err := domain.NewScrapeTarget(url)
	if err != nil {
		return nil
	}

	for _, scraper := range service.scrapers {
		if scraper.CanHandle(scrapeTarget) {
			scrapedSummary, scrapeError := safeScrape(scraper, context.Background(), scrapeTarget)
			if scrapeError != nil {
				log.Printf("Scraper failed for %s: %v", url, scrapeError)
				return nil
			}
			if scrapedSummary != nil {
				return scrapedSummary
			}
		}
	}

	return nil
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
