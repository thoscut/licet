package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// CacheConfig holds configuration for the cache middleware
type CacheConfig struct {
	DefaultTTL time.Duration
	MaxEntries int
	Enabled    bool
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		DefaultTTL: 30 * time.Second,
		MaxEntries: 1000,
		Enabled:    true,
	}
}

// cacheEntry represents a cached response
type cacheEntry struct {
	body        []byte
	contentType string
	statusCode  int
	headers     http.Header
	expiry      time.Time
}

// Cache is an in-memory cache for HTTP responses
type Cache struct {
	entries map[string]*cacheEntry
	mu      sync.RWMutex
	config  CacheConfig
	stopCh  chan struct{}
}

// NewCache creates a new cache instance
func NewCache(config CacheConfig) *Cache {
	c := &Cache{
		entries: make(map[string]*cacheEntry),
		config:  config,
		stopCh:  make(chan struct{}),
	}

	// Start cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Stop shuts down the cache cleanup goroutine
func (c *Cache) Stop() {
	close(c.stopCh)
}

// cleanupLoop periodically removes expired entries
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// cleanup removes expired entries
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiry) {
			delete(c.entries, key)
		}
	}
}

// Get retrieves a cached response
func (c *Cache) Get(key string) (*cacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiry) {
		return nil, false
	}

	return entry, true
}

// Set stores a response in the cache
func (c *Cache) Set(key string, entry *cacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Enforce max entries limit
	if len(c.entries) >= c.config.MaxEntries {
		// Remove oldest entries
		c.evictOldest()
	}

	c.entries[key] = entry
}

// evictOldest removes the oldest entries when cache is full
func (c *Cache) evictOldest() {
	// Simple eviction: remove entries closest to expiry
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestKey == "" || entry.expiry.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiry
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// Invalidate removes a specific key from the cache
func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// InvalidatePrefix removes all keys with a given prefix
func (c *Cache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.entries, key)
		}
	}
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// Stats returns cache statistics
func (c *Cache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"entries":     len(c.entries),
		"max_entries": c.config.MaxEntries,
		"default_ttl": c.config.DefaultTTL.String(),
		"enabled":     c.config.Enabled,
	}
}

// cachedResponseWriter wraps http.ResponseWriter to capture the response
type cachedResponseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func newCachedResponseWriter(w http.ResponseWriter) *cachedResponseWriter {
	return &cachedResponseWriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}
}

func (w *cachedResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *cachedResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// generateCacheKey creates a unique cache key from the request
func generateCacheKey(r *http.Request) string {
	// Include method, path, and query string
	data := r.Method + ":" + r.URL.Path + "?" + r.URL.RawQuery
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CacheMiddleware creates HTTP middleware for caching GET responses
func CacheMiddleware(cache *Cache, ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only cache GET requests
			if r.Method != http.MethodGet || !cache.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Check for cache-control: no-cache header
			if r.Header.Get("Cache-Control") == "no-cache" {
				next.ServeHTTP(w, r)
				return
			}

			key := generateCacheKey(r)

			// Try to get from cache
			if entry, found := cache.Get(key); found {
				log.WithField("path", r.URL.Path).Debug("Cache hit")
				w.Header().Set("X-Cache", "HIT")
				w.Header().Set("Content-Type", entry.contentType)
				for k, v := range entry.headers {
					w.Header()[k] = v
				}
				w.WriteHeader(entry.statusCode)
				w.Write(entry.body)
				return
			}

			// Cache miss - capture the response
			log.WithField("path", r.URL.Path).Debug("Cache miss")
			w.Header().Set("X-Cache", "MISS")

			crw := newCachedResponseWriter(w)
			next.ServeHTTP(crw, r)

			// Only cache successful responses
			if crw.statusCode >= 200 && crw.statusCode < 300 {
				entry := &cacheEntry{
					body:        crw.body.Bytes(),
					contentType: crw.Header().Get("Content-Type"),
					statusCode:  crw.statusCode,
					headers:     cloneHeaders(crw.Header()),
					expiry:      time.Now().Add(ttl),
				}
				cache.Set(key, entry)
			}
		})
	}
}

// cloneHeaders creates a copy of HTTP headers
func cloneHeaders(h http.Header) http.Header {
	clone := make(http.Header)
	for k, v := range h {
		// Skip certain headers that shouldn't be cached
		if k == "X-Cache" || k == "Date" {
			continue
		}
		clone[k] = append([]string(nil), v...)
	}
	return clone
}
