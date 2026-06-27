package infrastructure

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"
)

const (
	DefaultPerDomainMaxConcurrency = 5
	DefaultDomainCooldownDuration  = 30 * time.Second
	domainLimiterCleanupInterval   = 1 * time.Minute
)

type DomainRateLimiter struct {
	mu               sync.Mutex
	domains          map[string]*domainState
	maxPerDomain     int
	cooldownDuration time.Duration
	closed           chan struct{}
}

type domainState struct {
	sem          chan struct{}
	cooldownUntil time.Time
	refCount     int
}

func NewDomainRateLimiter(maxPerDomain int, cooldownDuration time.Duration) *DomainRateLimiter {
	if maxPerDomain <= 0 {
		maxPerDomain = DefaultPerDomainMaxConcurrency
	}
	if cooldownDuration <= 0 {
		cooldownDuration = DefaultDomainCooldownDuration
	}
	limiter := &DomainRateLimiter{
		domains:          make(map[string]*domainState),
		maxPerDomain:     maxPerDomain,
		cooldownDuration: cooldownDuration,
		closed:           make(chan struct{}),
	}
	go limiter.cleanupLoop()
	return limiter
}

func (l *DomainRateLimiter) Stop() {
	close(l.closed)
}

func ExtractHost(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse url: %w", err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("url has no host")
	}
	return parsed.Host, nil
}

func (l *DomainRateLimiter) Acquire(ctx context.Context, host string) error {
	l.mu.Lock()
	state, exists := l.domains[host]
	if !exists {
		state = &domainState{
			sem: make(chan struct{}, l.maxPerDomain),
		}
		l.domains[host] = state
	}
	if time.Now().Before(state.cooldownUntil) {
		remaining := time.Until(state.cooldownUntil).Round(time.Second)
		l.mu.Unlock()
		return fmt.Errorf("domain %s is rate limited (cooldown %s remaining)", host, remaining)
	}
	state.refCount++
	l.mu.Unlock()

	select {
	case state.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		l.mu.Lock()
		state.refCount--
		l.mu.Unlock()
		return ctx.Err()
	}
}

func (l *DomainRateLimiter) Release(host string) {
	l.mu.Lock()
	state, exists := l.domains[host]
	if !exists {
		l.mu.Unlock()
		return
	}
	state.refCount--
	if state.refCount < 0 {
		state.refCount = 0
	}
	<-state.sem
	l.mu.Unlock()
}

func (l *DomainRateLimiter) ReportRateLimited(host string) {
	l.mu.Lock()
	state, exists := l.domains[host]
	if !exists {
		state = &domainState{
			sem: make(chan struct{}, l.maxPerDomain),
		}
		l.domains[host] = state
	}
	state.cooldownUntil = time.Now().Add(l.cooldownDuration)
	l.mu.Unlock()
}

func (l *DomainRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(domainLimiterCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.closed:
			return
		}
	}
}

func (l *DomainRateLimiter) cleanup() {
	l.mu.Lock()
	now := time.Now()
	for host, state := range l.domains {
		if state.refCount == 0 && now.After(state.cooldownUntil) {
			delete(l.domains, host)
		}
	}
	l.mu.Unlock()
}
