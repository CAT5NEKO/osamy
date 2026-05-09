package infrastructure

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/user/osamy/internal/domain"
)

type RedisCacheRepository struct {
	redisClient *redis.Client
	expiration  time.Duration
}

func NewRedisCacheRepository(redisClient *redis.Client, expiration time.Duration) *RedisCacheRepository {
	return &RedisCacheRepository{
		redisClient: redisClient,
		expiration:  expiration,
	}
}

func (repository *RedisCacheRepository) Get(ctx context.Context, url string) (*domain.PageSummary, error) {
	serializedData, fetchError := repository.redisClient.Get(ctx, url).Result()
	if fetchError != nil {
		return nil, fetchError
	}

	var pageSummary domain.PageSummary
	if unmarshalError := json.Unmarshal([]byte(serializedData), &pageSummary); unmarshalError != nil {
		return nil, unmarshalError
	}

	pageSummary.Finalize()
	return &pageSummary, nil
}

func (repository *RedisCacheRepository) Set(ctx context.Context, url string, pageSummary *domain.PageSummary) error {
	serializedData, marshalError := json.Marshal(pageSummary)
	if marshalError != nil {
		return marshalError
	}

	return repository.redisClient.Set(ctx, url, serializedData, repository.expiration).Err()
}
