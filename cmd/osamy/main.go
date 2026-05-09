package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/user/osamy/internal/application"
	"github.com/user/osamy/internal/domain"
	"github.com/user/osamy/internal/infrastructure"
	"github.com/user/osamy/internal/interfaces"
)

func main() {
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == ""  {
		redisUrl = "localhost:34165"
	}

	var cacheRepository domain.CacheRepository

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisUrl,
	})

	pingContext, cancelPing := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelPing()

	if pingError := redisClient.Ping(pingContext).Err(); pingError != nil {
		log.Printf("Redis connection failed, fallback to in-memory cache: %v", pingError)
		cacheRepository = infrastructure.NewInMemoryCacheRepository(24 * time.Hour)
	} else {
		cacheRepository = infrastructure.NewRedisCacheRepository(redisClient, 24 * time.Hour)
	}

	scrapeTimeoutMs, parseError := strconv.Atoi(os.Getenv("SCRAPE_TIMEOUT_MS"))
	if parseError != nil {
		scrapeTimeoutMs = 10000
	}

	httpClient := &http.Client{
		Timeout:       time.Duration(scrapeTimeoutMs) * time.Millisecond,
		Transport:     infrastructure.NewSafeHttpTransport(),
		CheckRedirect: infrastructure.NewSafeRedirectPolicy(),
	}
	webFetcher := infrastructure.NewWebFetcher(httpClient)

	scrapers := []domain.ScraperDriver{
		infrastructure.NewYouTubeScraper(webFetcher),
		infrastructure.NewSpotifyScraper(webFetcher),
		infrastructure.NewTwitterScraper(webFetcher),
		infrastructure.NewNicoNicoScraper(webFetcher),
		infrastructure.NewBlueskyScraper(webFetcher),
		infrastructure.NewThreadsScraper(webFetcher),
		infrastructure.NewAmazonScraper(webFetcher),
		infrastructure.NewYodobashiScraper(webFetcher),
		infrastructure.NewNitoriScraper(webFetcher),
		infrastructure.NewGeneralScraper(webFetcher),
	}

	maxConcurrency, parseError := strconv.Atoi(os.Getenv("MAX_CONCURRENCY"))
	if parseError != nil {
		maxConcurrency = 10
	}

	summaryService := application.NewSummaryApplicationService(scrapers, cacheRepository, maxConcurrency)
	summaryHandler := interfaces.NewSummaryHandler(summaryService)
	healthHandler := interfaces.NewHealthHandler()

	rateLimiter := interfaces.NewRateLimiter(60, 1*time.Minute)

	http.Handle("/", rateLimiter.Middleware(summaryHandler))
	http.Handle("/health", healthHandler)

	host := os.Getenv("HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	address := host + ":" + port
	log.Printf("Server starting on %s", address)
	if listenError := http.ListenAndServe(address, nil); listenError != nil {
		log.Fatalf("Server failed: %v", listenError)
	}
}
