package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// RequestsPerMinute is the maximum number of requests allowed per minute per IP
	RequestsPerMinute int
	// BurstSize is the maximum burst size allowed
	BurstSize int
	// Enabled controls whether rate limiting is active
	Enabled bool
	// WhitelistedIPs are exempt from rate limiting
	WhitelistedIPs []string
	// WhitelistedPaths are exempt from rate limiting (e.g., health checks)
	WhitelistedPaths []string
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 100,
		BurstSize:         20,
		Enabled:           true,
		WhitelistedIPs:    []string{"127.0.0.1", "::1"},
		WhitelistedPaths:  []string{"/api/v1/health", "/static/"},
	}
}

// rateLimitEntry tracks request counts for an IP
type rateLimitEntry struct {
	tokens     float64
	lastUpdate time.Time
}

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	entries          map[string]*rateLimitEntry
	mu               sync.RWMutex
	config           RateLimitConfig
	tokensPerSecond  float64
	whitelistedIPs   map[string]bool
	whitelistedPaths []string
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	whitelistedIPs := make(map[string]bool)
	for _, ip := range config.WhitelistedIPs {
		whitelistedIPs[ip] = true
	}

	rl := &RateLimiter{
		entries:          make(map[string]*rateLimitEntry),
		config:           config,
		tokensPerSecond:  float64(config.RequestsPerMinute) / 60.0,
		whitelistedIPs:   whitelistedIPs,
		whitelistedPaths: config.WhitelistedPaths,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically removes stale entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes entries that haven't been accessed in a while
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, entry := range rl.entries {
		if entry.lastUpdate.Before(cutoff) {
			delete(rl.entries, ip)
		}
	}
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) (bool, int, time.Time) {
	if !rl.config.Enabled {
		return true, rl.config.BurstSize, time.Time{}
	}

	// Check whitelist
	if rl.whitelistedIPs[ip] {
		return true, rl.config.BurstSize, time.Time{}
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	entry, exists := rl.entries[ip]

	if !exists {
		entry = &rateLimitEntry{
			tokens:     float64(rl.config.BurstSize),
			lastUpdate: now,
		}
		rl.entries[ip] = entry
	}

	// Add tokens based on time elapsed
	elapsed := now.Sub(entry.lastUpdate).Seconds()
	entry.tokens += elapsed * rl.tokensPerSecond

	// Cap tokens at burst size
	if entry.tokens > float64(rl.config.BurstSize) {
		entry.tokens = float64(rl.config.BurstSize)
	}

	entry.lastUpdate = now

	// Check if we have tokens available
	if entry.tokens >= 1.0 {
		entry.tokens -= 1.0
		return true, int(entry.tokens), time.Time{}
	}

	// Calculate when the next token will be available
	waitTime := (1.0 - entry.tokens) / rl.tokensPerSecond
	retryAfter := now.Add(time.Duration(waitTime * float64(time.Second)))

	return false, 0, retryAfter
}

// isWhitelistedPath checks if the request path is whitelisted
func (rl *RateLimiter) isWhitelistedPath(path string) bool {
	for _, wp := range rl.whitelistedPaths {
		if len(path) >= len(wp) && path[:len(wp)] == wp {
			return true
		}
	}
	return false
}

// Stats returns rate limiter statistics
func (rl *RateLimiter) Stats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"tracked_ips":         len(rl.entries),
		"requests_per_minute": rl.config.RequestsPerMinute,
		"burst_size":          rl.config.BurstSize,
		"enabled":             rl.config.Enabled,
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// Strip port if present
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
		if addr[i] == ']' {
			// IPv6 address without port
			break
		}
	}
	return addr
}

// RateLimitMiddleware creates HTTP middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip rate limiting for whitelisted paths
			if limiter.isWhitelistedPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			ip := getClientIP(r)
			allowed, remaining, retryAfter := limiter.Allow(ip)

			// Always set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.config.RequestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if !allowed {
				log.WithFields(log.Fields{
					"ip":          ip,
					"path":        r.URL.Path,
					"retry_after": retryAfter.Format(time.RFC1123),
				}).Warn("Rate limit exceeded")

				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(retryAfter.Unix(), 10))
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(retryAfter).Seconds())+1))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)

				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":       "rate limit exceeded",
					"message":     "Too many requests. Please try again later.",
					"retry_after": int(time.Until(retryAfter).Seconds()) + 1,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
