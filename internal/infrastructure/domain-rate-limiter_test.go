package infrastructure

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestDomainRateLimiterAcquireRelease(t *testing.T) {
	limiter := NewDomainRateLimiter(2, time.Minute)
	defer limiter.Stop()

	host := "example.com"

	err := limiter.Acquire(context.Background(), host)
	if err != nil {
		t.Fatalf("expected Acquire to succeed: %v", err)
	}

	err = limiter.Acquire(context.Background(), host)
	if err != nil {
		t.Fatalf("expected second Acquire to succeed: %v", err)
	}

	limiter.Release(host)
	limiter.Release(host)
}

func TestDomainRateLimiterCooldown(t *testing.T) {
	limiter := NewDomainRateLimiter(5, 100*time.Millisecond)
	defer limiter.Stop()

	host := "example.com"

	limiter.ReportRateLimited(host)

	err := limiter.Acquire(context.Background(), host)
	if err == nil {
		t.Fatalf("expected Acquire to fail during cooldown")
	}
}

func TestDomainRateLimiterCooldownExpires(t *testing.T) {
	limiter := NewDomainRateLimiter(5, 50*time.Millisecond)
	defer limiter.Stop()

	host := "example.com"

	limiter.ReportRateLimited(host)

	time.Sleep(100 * time.Millisecond)

	err := limiter.Acquire(context.Background(), host)
	if err != nil {
		t.Fatalf("expected Acquire to succeed after cooldown: %v", err)
	}
	limiter.Release(host)
}

func TestDomainRateLimiterMultipleHosts(t *testing.T) {
	limiter := NewDomainRateLimiter(2, time.Minute)
	defer limiter.Stop()

	err := limiter.Acquire(context.Background(), "host-a.com")
	if err != nil {
		t.Fatalf("expected Acquire to succeed: %v", err)
	}

	err = limiter.Acquire(context.Background(), "host-b.com")
	if err != nil {
		t.Fatalf("expected Acquire to succeed: %v", err)
	}

	limiter.Release("host-a.com")
	limiter.Release("host-b.com")
}

func TestDomainRateLimiterSelectiveCooldown(t *testing.T) {
	limiter := NewDomainRateLimiter(5, time.Minute)
	defer limiter.Stop()

	limiter.ReportRateLimited("badhost.com")

	err := limiter.Acquire(context.Background(), "badhost.com")
	if err == nil {
		t.Fatalf("expected Acquire to fail for badhost.com")
	}

	err = limiter.Acquire(context.Background(), "goodhost.com")
	if err != nil {
		t.Fatalf("expected Acquire to succeed for goodhost.com: %v", err)
	}
	limiter.Release("goodhost.com")
}

func TestDomainRateLimiterConcurrentLimit(t *testing.T) {
	limiter := NewDomainRateLimiter(3, time.Minute)
	defer limiter.Stop()

	host := "example.com"

	for i := 0; i < 3; i++ {
		err := limiter.Acquire(context.Background(), host)
		if err != nil {
			t.Fatalf("expected Acquire %d to succeed: %v", i, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := limiter.Acquire(ctx, host)
	if err == nil {
		t.Fatalf("expected Acquire to fail when at concurrency limit")
	}

	for i := 0; i < 3; i++ {
		limiter.Release(host)
	}
}

func TestDomainRateLimiterConcurrentAccess(t *testing.T) {
	limiter := NewDomainRateLimiter(10, time.Minute)
	defer limiter.Stop()

	host := "example.com"
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := limiter.Acquire(context.Background(), host)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			time.Sleep(10 * time.Millisecond)
			limiter.Release(host)
		}()
	}

	wg.Wait()
}

func TestExtractHost(t *testing.T) {
	cases := []struct {
		input string
		host  string
	}{
		{input: "https://example.com/page", host: "example.com"},
		{input: "http://www.example.com", host: "www.example.com"},
		{input: "https://amazon.co.jp/dp/12345", host: "amazon.co.jp"},
		{input: "https://youtu.be/abc123", host: "youtu.be"},
	}

	for _, tc := range cases {
		host, err := ExtractHost(tc.input)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tc.input, err)
		}
		if host != tc.host {
			t.Fatalf("expected %s, got %s", tc.host, host)
		}
	}
}

func TestExtractHostInvalidURL(t *testing.T) {
	_, err := ExtractHost("not-a-url")
	if err == nil {
		t.Fatalf("expected error for invalid URL")
	}
}
