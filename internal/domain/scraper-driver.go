package domain

import "context"

type ScraperDriver interface {
	CanHandle(target *ScrapeTarget) bool
	Scrape(ctx context.Context, target *ScrapeTarget) (*PageSummary, error)
}
