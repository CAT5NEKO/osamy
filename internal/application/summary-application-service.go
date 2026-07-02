package 	application

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"runtime/debug"
	"sync"

	"github.com/user/osamy/internal/domain"
)

type DomainLimiter interface {
	Acquire(ctx context.Context, host string) error
	Release(host string)
	ReportRateLimited(host string)
}

type inflightEntry struct {
	done   chan struct{}
	result *domain.PageSummary
}

type SummaryApplicationService struct {
	scrapers      []domain.ScraperDriver
	cache         domain.CacheRepository
	semaphore     chan struct{}
	inflight      sync.Map
	domainLimiter DomainLimiter
}

func NewSummaryApplicationService(scrapers []domain.ScraperDriver, cache domain.CacheRepository, maxConcurrency int, domainLimiter DomainLimiter) *SummaryApplicationService {
	return &SummaryApplicationService{
		scrapers:      scrapers,
		cache:         cache,
		semaphore:     make(chan struct{}, maxConcurrency),
		domainLimiter: domainLimiter,
	}
}

func (service *SummaryApplicationService) GetSummary(ctx context.Context, urlStr string) (*domain.PageSummary, error) {
	cachedSummary, cacheError := service.cache.Get(ctx, urlStr)
	if cacheError == nil && cachedSummary != nil {
		return cachedSummary, nil
	}
	if cacheError != nil {
		log.Printf("Cache access failed: %v", cacheError)
	}

	entry := &inflightEntry{done: make(chan struct{})}
	existing, loaded := service.inflight.LoadOrStore(urlStr, entry)
	if loaded {
		existingEntry := existing.(*inflightEntry)
		select {
		case <-existingEntry.done:
			return existingEntry.result, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	defer service.inflight.Delete(urlStr)

	if service.domainLimiter != nil {
		host, extractErr := extractHost(urlStr)
		if extractErr != nil {
			return nil, extractErr
		}

		if acquireErr := service.domainLimiter.Acquire(ctx, host); acquireErr != nil {
			return nil, acquireErr
		}
		defer service.domainLimiter.Release(host)
	}

	select {
	case service.semaphore <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-service.semaphore }()

	result := service.fetchURL(ctx, urlStr)

	entry.result = result
	close(entry.done)

	if result != nil {
		_ = service.cache.Set(context.Background(), urlStr, result)
	}
	return result, nil
}

func extractHost(rawURL string) (string, error) {
	parsed, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", fmt.Errorf("failed to parse url: %w", parseErr)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("url has no host")
	}
	return parsed.Host, nil
}

func (service *SummaryApplicationService) fetchURL(ctx context.Context, url string) *domain.PageSummary {
	scrapeTarget, err := domain.NewScrapeTarget(url)
	if err != nil {
		return nil
	}

	for _, scraper := range service.scrapers {
		if scraper.CanHandle(scrapeTarget) {
			scrapedSummary, scrapeError := safeScrape(scraper, ctx, scrapeTarget)
			if scrapeError != nil {
				log.Printf("Scraper failed for %s: %v, trying next scraper", url, scrapeError)
				continue
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
