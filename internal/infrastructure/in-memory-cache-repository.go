package infrastructure

import (
	"context"
	"sync"
	"time"

	"github.com/user/osamy/internal/domain"
)

type cacheItem struct {
	pageSummary *domain.PageSummary
	expiresAt   time.Time
}

type InMemoryCacheRepository struct {
	cacheStorage sync.Map
	defaultTtl   time.Duration
}

func NewInMemoryCacheRepository(defaultTtl time.Duration) *InMemoryCacheRepository {
	return &InMemoryCacheRepository{
		defaultTtl: defaultTtl,
	}
}

func (repository *InMemoryCacheRepository) Get(ctx context.Context, targetUrl string) (*domain.PageSummary, error) {
	storedValue, isFound := repository.cacheStorage.Load(targetUrl)
	if !isFound {
		return nil, nil
	}

	item := storedValue.(cacheItem)
	if time.Now().After(item.expiresAt) {
		repository.cacheStorage.Delete(targetUrl)
		return nil, nil
	}

	item.pageSummary.Finalize()
	return item.pageSummary, nil
}

func (repository *InMemoryCacheRepository) Set(ctx context.Context, targetUrl string, pageSummary *domain.PageSummary) error {
	repository.cacheStorage.Store(targetUrl, cacheItem{
		pageSummary: pageSummary,
		expiresAt:   time.Now().Add(repository.defaultTtl),
	})
	return nil
}
