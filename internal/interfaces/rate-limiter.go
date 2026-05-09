package interfaces

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiterEntry struct {
	RequestCount int
	WindowStart  time.Time
}

type RateLimiter struct {
	maxRequestsPerWindow int
	windowDuration       time.Duration
	entries              map[string]*RateLimiterEntry
	mutex                sync.Mutex
}

func NewRateLimiter(maxRequestsPerWindow int, windowDuration time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		maxRequestsPerWindow: maxRequestsPerWindow,
		windowDuration:       windowDuration,
		entries:              make(map[string]*RateLimiterEntry),
	}

	go limiter.cleanupExpiredEntries()

	return limiter
}

func (limiter *RateLimiter) IsAllowed(clientIdentifier string) bool {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	now := time.Now()
	entry, exists := limiter.entries[clientIdentifier]

	if !exists || now.Sub(entry.WindowStart) > limiter.windowDuration {
		limiter.entries[clientIdentifier] = &RateLimiterEntry{
			RequestCount: 1,
			WindowStart:  now,
		}
		return true
	}

	if entry.RequestCount >= limiter.maxRequestsPerWindow {
		return false
	}

	entry.RequestCount++
	return true
}

func (limiter *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		clientIp := extractClientIp(request)

		if !limiter.IsAllowed(clientIp) {
			http.Error(writer, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

func (limiter *RateLimiter) cleanupExpiredEntries() {
	ticker := time.NewTicker(limiter.windowDuration)
	defer ticker.Stop()

	for range ticker.C {
		limiter.mutex.Lock()
		now := time.Now()
		for key, entry := range limiter.entries {
			if now.Sub(entry.WindowStart) > limiter.windowDuration {
				delete(limiter.entries, key)
			}
		}
		limiter.mutex.Unlock()
	}
}

func extractClientIp(request *http.Request) string {
	forwarded := request.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	realIp := request.Header.Get("X-Real-IP")
	if realIp != "" {
		return realIp
	}

	host, _, splitError := net.SplitHostPort(request.RemoteAddr)
	if splitError != nil {
		return request.RemoteAddr
	}
	return host
}
