package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter_AllowsRequests(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		Enabled:           true,
	})
	defer rl.Stop()

	allowed, remaining, _ := rl.Allow("192.168.1.1")
	if !allowed {
		t.Fatal("expected first request to be allowed")
	}
	if remaining != 9 {
		t.Errorf("expected 9 remaining tokens, got %d", remaining)
	}
}

func TestRateLimiter_BurstExhaustion(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         3,
		Enabled:           true,
	})
	defer rl.Stop()

	// Exhaust burst
	for i := 0; i < 3; i++ {
		allowed, _, _ := rl.Allow("192.168.1.1")
		if !allowed {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}

	// Next request should be denied
	allowed, _, retryAfter := rl.Allow("192.168.1.1")
	if allowed {
		t.Fatal("expected request to be denied after burst exhaustion")
	}
	if retryAfter.IsZero() {
		t.Error("expected non-zero retry-after time")
	}
}

func TestRateLimiter_WhitelistedIP(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         1,
		Enabled:           true,
		WhitelistedIPs:    []string{"10.0.0.1"},
	})
	defer rl.Stop()

	// Exhaust burst for whitelisted IP -- should always be allowed
	for i := 0; i < 5; i++ {
		allowed, _, _ := rl.Allow("10.0.0.1")
		if !allowed {
			t.Fatalf("whitelisted IP should always be allowed, failed on request %d", i+1)
		}
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		RequestsPerMinute: 1,
		BurstSize:         1,
		Enabled:           false,
	})
	defer rl.Stop()

	for i := 0; i < 10; i++ {
		allowed, _, _ := rl.Allow("192.168.1.1")
		if !allowed {
			t.Fatal("disabled rate limiter should always allow")
		}
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         1,
		Enabled:           true,
	})
	defer rl.Stop()

	// Exhaust burst for IP 1
	rl.Allow("192.168.1.1")
	allowed, _, _ := rl.Allow("192.168.1.1")
	if allowed {
		t.Fatal("expected IP 1 to be rate limited")
	}

	// IP 2 should still be allowed
	allowed, _, _ = rl.Allow("192.168.1.2")
	if !allowed {
		t.Fatal("expected IP 2 to be allowed (separate bucket)")
	}
}

func TestRateLimitMiddleware_SetsHeaders(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		RequestsPerMinute: 100,
		BurstSize:         10,
		Enabled:           true,
	})
	defer rl.Stop()

	handler := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
	if rr.Header().Get("X-RateLimit-Limit") != "100" {
		t.Errorf("expected X-RateLimit-Limit=100, got %q", rr.Header().Get("X-RateLimit-Limit"))
	}
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("expected X-RateLimit-Remaining header to be set")
	}
}

func TestRateLimitMiddleware_WhitelistedPath(t *testing.T) {
	rl := NewRateLimiter(RateLimitConfig{
		RequestsPerMinute: 1,
		BurstSize:         1,
		Enabled:           true,
		WhitelistedPaths:  []string{"/api/v1/health"},
	})
	defer rl.Stop()

	handler := RateLimitMiddleware(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make many requests to a whitelisted path
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("whitelisted path request %d returned status %d", i+1, rr.Code)
		}
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		want       string
	}{
		{"X-Forwarded-For", "10.0.0.1, 10.0.0.2", "", "192.168.1.1:1234", "10.0.0.1"},
		{"X-Real-IP", "", "10.0.0.1", "192.168.1.1:1234", "10.0.0.1"},
		{"RemoteAddr with port", "", "", "192.168.1.1:1234", "192.168.1.1"},
		{"RemoteAddr without port", "", "", "192.168.1.1", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			req.RemoteAddr = tt.remoteAddr

			got := getClientIP(req)
			if got != tt.want {
				t.Errorf("getClientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
