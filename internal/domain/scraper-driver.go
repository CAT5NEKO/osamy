package domain

import "context"

type ScraperDriver interface {
	CanHandle(url string) bool
	Scrape(ctx context.Context, url string) (*PageSummary, error)
}
