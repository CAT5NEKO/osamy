package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/user/osamy/internal/application"
	"github.com/user/osamy/internal/domain"
	"github.com/user/osamy/internal/infrastructure"
	"github.com/user/osamy/internal/interfaces"
)

func main() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:34165"
	}

	var cacheRepository domain.CacheRepository

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})

	pingContext, cancelPing := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelPing()

	if pingError := redisClient.Ping(pingContext).Err(); pingError != nil {
		log.Printf("Redis connection failed, fallback to in-memory cache: %v", pingError)
		cacheRepository = infrastructure.NewInMemoryCacheRepository(24 * time.Hour)
	} else {
		cacheRepository = infrastructure.NewRedisCacheRepository(redisClient, 24*time.Hour)
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

	rateLimiter := interfaces.NewRateLimiter(600, 1*time.Minute)

	mux := http.NewServeMux()
	mux.Handle("/", rateLimiter.Middleware(summaryHandler))
	mux.Handle("/health", healthHandler)

	host := os.Getenv("HOST")
	if host == "" {
		host = ""
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	address := host + ":" + port
	pprofAddr := os.Getenv("PPROF_ADDR")
	pprofServer, startError := infrastructure.StartPprofServer(pprofAddr)
	if startError != nil {
		log.Printf("pprof server stopped: %v", startError)
	}

	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadTimeout:       60 * time.Second,
		ReadHeaderTimeout: 60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server starting on %s", address)
		if listenError := server.ListenAndServe(); listenError != nil && !errors.Is(listenError, http.ErrServerClosed) {
			log.Printf("Server failed: %v", listenError)
		}
	}()

	<-quit
	log.Printf("Shutting down server...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if shutdownError := server.Shutdown(shutdownCtx); shutdownError != nil {
		log.Printf("Server shutdown error: %v", shutdownError)
	}

	if pprofServer != nil {
		_ = pprofServer.Close()
	}

	log.Printf("Server stopped")
}
