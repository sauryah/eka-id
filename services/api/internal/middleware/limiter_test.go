package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sauryah/eka-id/services/api/internal/middleware"
)

func TestMemoryRateLimiter_AllowsWithinLimit(t *testing.T) {
	limiter := middleware.NewMemoryRateLimiter()
	ctx := context.Background()
	key := "test-ip-1"

	// Limit: 3 requests per 2 seconds
	for i := 1; i <= 3; i++ {
		allowed, remaining, _ := limiter.Allow(ctx, key, 3, 2*time.Second)
		if !allowed {
			t.Fatalf("Request %d was unexpectedly rate limited", i)
		}
		expectedRemaining := 3 - i
		if remaining != expectedRemaining {
			t.Fatalf("Expected remaining %d, got %d", expectedRemaining, remaining)
		}
	}

	// 4th request must be rejected
	allowed, remaining, resetAfter := limiter.Allow(ctx, key, 3, 2*time.Second)
	if allowed {
		t.Fatal("4th request should have been rejected by rate limiter")
	}
	if remaining != 0 {
		t.Fatalf("Expected remaining 0, got %d", remaining)
	}
	if resetAfter <= 0 {
		t.Fatalf("Expected positive resetAfter duration, got %v", resetAfter)
	}
}

func TestHybridRateLimiter_GracefulMemoryFallback(t *testing.T) {
	// Point to non-existent Redis to test graceful fallback
	limiter := middleware.NewHybridRateLimiter("127.0.0.1", "65432", "")
	ctx := context.Background()
	key := "fallback-client"

	// Should not panic, error out, or block
	for i := 1; i <= 2; i++ {
		allowed, _, _ := limiter.Allow(ctx, key, 2, time.Second)
		if !allowed {
			t.Fatalf("Request %d failed under memory fallback", i)
		}
	}

	// 3rd should be blocked
	allowed, _, _ := limiter.Allow(ctx, key, 2, time.Second)
	if allowed {
		t.Fatal("3rd request should be blocked under fallback memory limiter")
	}
}

func TestRateLimitMiddleware_HeadersAndHTTP429(t *testing.T) {
	limiter := middleware.NewHybridRateLimiter("", "", "")
	handler := limiter.Limit(2, time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	// Request 1
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("Request 1 expected 200, got %d", w1.Code)
	}
	if w1.Header().Get("X-RateLimit-Limit") != "2" {
		t.Fatalf("Expected X-RateLimit-Limit 2, got %s", w1.Header().Get("X-RateLimit-Limit"))
	}
	if w1.Header().Get("X-RateLimit-Remaining") != "1" {
		t.Fatalf("Expected X-RateLimit-Remaining 1, got %s", w1.Header().Get("X-RateLimit-Remaining"))
	}

	// Request 2
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "10.0.0.1:1234"
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("Request 2 expected 200, got %d", w2.Code)
	}
	if w2.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Fatalf("Expected X-RateLimit-Remaining 0, got %s", w2.Header().Get("X-RateLimit-Remaining"))
	}

	// Request 3 (Over limit -> 429)
	req3 := httptest.NewRequest("GET", "/test", nil)
	req3.RemoteAddr = "10.0.0.1:1234"
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("Request 3 expected 429, got %d", w3.Code)
	}
	if w3.Header().Get("Retry-After") == "" {
		t.Fatal("Expected Retry-After header on 429 response")
	}
}
