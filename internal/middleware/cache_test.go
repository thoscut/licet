package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Minute,
		MaxEntries: 100,
		Enabled:    true,
	})
	defer cache.Stop()

	entry := &cacheEntry{
		body:        []byte(`{"test": true}`),
		contentType: "application/json",
		statusCode:  200,
		headers:     http.Header{},
		expiry:      time.Now().Add(time.Minute),
	}
	cache.Set("key1", entry)

	got, found := cache.Get("key1")
	if !found {
		t.Fatal("expected cache hit")
	}
	if string(got.body) != `{"test": true}` {
		t.Errorf("got body %q, want %q", string(got.body), `{"test": true}`)
	}
}

func TestCache_MissOnExpired(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Millisecond,
		MaxEntries: 100,
		Enabled:    true,
	})
	defer cache.Stop()

	entry := &cacheEntry{
		body:   []byte("data"),
		expiry: time.Now().Add(-time.Second), // Already expired
	}
	cache.Set("key1", entry)

	_, found := cache.Get("key1")
	if found {
		t.Fatal("expected cache miss on expired entry")
	}
}

func TestCache_MissOnNonexistent(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Minute,
		MaxEntries: 100,
		Enabled:    true,
	})
	defer cache.Stop()

	_, found := cache.Get("nonexistent")
	if found {
		t.Fatal("expected cache miss on nonexistent key")
	}
}

func TestCache_Invalidate(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Minute,
		MaxEntries: 100,
		Enabled:    true,
	})
	defer cache.Stop()

	entry := &cacheEntry{body: []byte("data"), expiry: time.Now().Add(time.Minute)}
	cache.Set("key1", entry)
	cache.Invalidate("key1")

	_, found := cache.Get("key1")
	if found {
		t.Fatal("expected cache miss after invalidation")
	}
}

func TestCache_InvalidatePrefix(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Minute,
		MaxEntries: 100,
		Enabled:    true,
	})
	defer cache.Stop()

	entry := &cacheEntry{body: []byte("data"), expiry: time.Now().Add(time.Minute)}
	cache.Set("prefix:key1", entry)
	cache.Set("prefix:key2", entry)
	cache.Set("other:key3", entry)

	cache.InvalidatePrefix("prefix:")

	if _, found := cache.Get("prefix:key1"); found {
		t.Error("expected cache miss for prefix:key1")
	}
	if _, found := cache.Get("prefix:key2"); found {
		t.Error("expected cache miss for prefix:key2")
	}
	if _, found := cache.Get("other:key3"); !found {
		t.Error("expected cache hit for other:key3")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Minute,
		MaxEntries: 100,
		Enabled:    true,
	})
	defer cache.Stop()

	entry := &cacheEntry{body: []byte("data"), expiry: time.Now().Add(time.Minute)}
	cache.Set("key1", entry)
	cache.Set("key2", entry)
	cache.Clear()

	stats := cache.Stats()
	if stats["entries"].(int) != 0 {
		t.Errorf("expected 0 entries after clear, got %v", stats["entries"])
	}
}

func TestCache_MaxEntries(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Minute,
		MaxEntries: 2,
		Enabled:    true,
	})
	defer cache.Stop()

	entry := &cacheEntry{body: []byte("data"), expiry: time.Now().Add(time.Minute)}
	cache.Set("key1", entry)
	cache.Set("key2", entry)
	cache.Set("key3", entry) // Should evict oldest

	stats := cache.Stats()
	count := stats["entries"].(int)
	if count > 2 {
		t.Errorf("expected at most 2 entries, got %d", count)
	}
}

func TestCacheMiddleware_CachesGETResponse(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Minute,
		MaxEntries: 100,
		Enabled:    true,
	})
	defer cache.Stop()

	callCount := 0
	handler := CacheMiddleware(cache, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result": "ok"}`))
	}))

	// First request - cache miss
	req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Cache") != "MISS" {
		t.Errorf("expected X-Cache=MISS, got %q", rr.Header().Get("X-Cache"))
	}

	// Second request - cache hit
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("X-Cache") != "HIT" {
		t.Errorf("expected X-Cache=HIT, got %q", rr.Header().Get("X-Cache"))
	}

	if callCount != 1 {
		t.Errorf("handler should be called once, was called %d times", callCount)
	}
}

func TestCacheMiddleware_SkipsPOST(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Minute,
		MaxEntries: 100,
		Enabled:    true,
	})
	defer cache.Stop()

	callCount := 0
	handler := CacheMiddleware(cache, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte("ok"))
	}))

	// POST requests should not be cached
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/servers", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	if callCount != 3 {
		t.Errorf("handler should be called 3 times for POST, was called %d times", callCount)
	}
}

func TestCacheMiddleware_SkipsNoCacheHeader(t *testing.T) {
	cache := NewCache(CacheConfig{
		DefaultTTL: time.Minute,
		MaxEntries: 100,
		Enabled:    true,
	})
	defer cache.Stop()

	callCount := 0
	handler := CacheMiddleware(cache, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte("ok"))
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/servers", nil)
		req.Header.Set("Cache-Control", "no-cache")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}

	if callCount != 2 {
		t.Errorf("handler should be called twice with no-cache, was called %d times", callCount)
	}
}
