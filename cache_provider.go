package klayengo

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"
)

// DefaultCacheProvider wraps the existing Cache interface to provide CacheProvider functionality.
type DefaultCacheProvider struct {
	cache Cache
	ttl   time.Duration
}

// NewDefaultCacheProvider creates a CacheProvider that wraps an existing Cache.
func NewDefaultCacheProvider(cache Cache, ttl time.Duration) CacheProvider {
	return &DefaultCacheProvider{
		cache: cache,
		ttl:   ttl,
	}
}

// Get retrieves a cached response.
func (cp *DefaultCacheProvider) Get(ctx context.Context, key string) (*http.Response, bool) {
	entry, found := cp.cache.Get(key)
	if !found {
		return nil, false
	}

	// Create response from cache entry
	resp := &http.Response{
		StatusCode: entry.StatusCode,
		Header:     entry.Header,
		Body:       io.NopCloser(bytes.NewReader(entry.Body)),
	}

	return resp, true
}

// Set stores a response in the cache.
func (cp *DefaultCacheProvider) Set(ctx context.Context, key string, resp *http.Response, ttl time.Duration) {
	// Read response body only if it hasn't been read already
	var body []byte
	var err error

	if resp.Body != nil {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		_ = resp.Body.Close()

		// Create a new body for downstream consumption
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	// Create cache entry
	entry := &CacheEntry{
		Response:   resp,
		Body:       body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
	}

	// Use provided TTL or default
	if ttl == 0 {
		ttl = cp.ttl
	}

	cp.cache.Set(key, entry, ttl)
}

// Invalidate removes a cached response.
func (cp *DefaultCacheProvider) Invalidate(ctx context.Context, key string) {
	cp.cache.Delete(key)
}

// HTTPSemanticsCacheProvider implements CacheProvider with full HTTP semantics support.
type HTTPSemanticsCacheProvider struct {
	cache      Cache
	defaultTTL time.Duration
	mode       CacheMode
}

// NewHTTPSemanticsCacheProvider creates a CacheProvider with HTTP semantics support.
func NewHTTPSemanticsCacheProvider(cache Cache, defaultTTL time.Duration, mode CacheMode) CacheProvider {
	return &HTTPSemanticsCacheProvider{
		cache:      cache,
		defaultTTL: defaultTTL,
		mode:       mode,
	}
}

// Get retrieves a cached response with HTTP semantics.
func (cp *HTTPSemanticsCacheProvider) Get(ctx context.Context, key string) (*http.Response, bool) {
	entry, found := cp.cache.Get(key)
	if !found {
		return nil, false
	}

	now := time.Now()

	// Check if entry is expired
	if now.After(entry.ExpiresAt) {
		// In SWR mode, serve stale if within stale window
		if cp.mode == SWR && entry.StaleAt != nil && now.Before(*entry.StaleAt) {
			entry.IsStale = true
			return cp.createResponseFromEntry(entry), true
		}

		// Entry is expired and not servable
		cp.cache.Delete(key)
		return nil, false
	}

	// Entry is fresh
	return cp.createResponseFromEntry(entry), true
}

// Set stores a response with HTTP cache semantics.
func (cp *HTTPSemanticsCacheProvider) Set(ctx context.Context, key string, resp *http.Response, ttl time.Duration) {
	// Read response body only if it hasn't been read already
	var body []byte
	var err error

	if resp.Body != nil {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		_ = resp.Body.Close()

		// Create a new body for downstream consumption
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	// Create enhanced cache entry with HTTP metadata
	entry := createEnhancedCacheEntry(resp, body, time.Now())

	// Use HTTP-derived TTL if available, otherwise use provided/default TTL
	if entry.ExpiresAt.IsZero() {
		if ttl == 0 {
			ttl = cp.defaultTTL
		}
		entry.ExpiresAt = time.Now().Add(ttl)
	}

	cp.cache.Set(key, entry, time.Until(entry.ExpiresAt))
}

// Invalidate removes a cached response.
func (cp *HTTPSemanticsCacheProvider) Invalidate(ctx context.Context, key string) {
	cp.cache.Delete(key)
}

// createResponseFromEntry reconstructs an HTTP response from a cache entry.
func (cp *HTTPSemanticsCacheProvider) createResponseFromEntry(entry *CacheEntry) *http.Response {
	resp := &http.Response{
		StatusCode: entry.StatusCode,
		Header:     entry.Header.Clone(),
		Body:       io.NopCloser(bytes.NewReader(entry.Body)),
	}

	// Add cache status headers for debugging
	if entry.IsStale {
		resp.Header.Set("X-Cache-Status", "stale")
	} else {
		resp.Header.Set("X-Cache-Status", "hit")
	}

	return resp
}
