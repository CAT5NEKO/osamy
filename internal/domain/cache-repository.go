package domain

import "context"

type CacheRepository interface {
	Get(ctx context.Context, url string) (*PageSummary, error)
	Set(ctx context.Context, url string, summary *PageSummary) error
}
