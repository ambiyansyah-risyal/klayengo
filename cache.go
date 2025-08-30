package klayengo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// InMemoryCache is a simple in-memory cache implementation
type InMemoryCache struct {
	mu    sync.RWMutex
	store map[string]*CacheEntry
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{
		store: make(map[string]*CacheEntry),
	}
}

// Get retrieves a cached entry
func (c *InMemoryCache) Get(key string) (*CacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.store[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, remove it
		delete(c.store, key)
		return nil, false
	}

	return entry, true
}

// Set stores a cache entry
func (c *InMemoryCache) Set(key string, entry *CacheEntry, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry.ExpiresAt = time.Now().Add(ttl)
	c.store[key] = entry
}

// Delete removes a cache entry
func (c *InMemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.store, key)
}

// Clear removes all cache entries
func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store = make(map[string]*CacheEntry)
}

// createResponseFromCache creates an HTTP response from a cached entry
func (c *Client) createResponseFromCache(entry *CacheEntry) *http.Response {
	resp := &http.Response{
		StatusCode: entry.StatusCode,
		Header:     entry.Header,
		Body:       io.NopCloser(bytes.NewReader(entry.Body)),
	}
	return resp
}

// createCacheEntry creates a cache entry from an HTTP response
func (c *Client) createCacheEntry(resp *http.Response) *CacheEntry {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Restore the body for the caller
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return &CacheEntry{
		Response:   resp,
		Body:       body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
	}
}

// DefaultCacheKeyFunc generates a cache key from the request
func DefaultCacheKeyFunc(req *http.Request) string {
	return fmt.Sprintf("%s:%s", req.Method, req.URL.String())
}

// DefaultCacheCondition determines if a request should be cached
func DefaultCacheCondition(req *http.Request) bool {
	// Only cache GET requests by default
	return req.Method == "GET"
}

// shouldCacheRequest determines if a request should be cached
func (c *Client) shouldCacheRequest(req *http.Request) bool {
	// Cache must be enabled
	if c.cache == nil {
		return false
	}

	// Check context-based cache control
	if cacheControl, ok := req.Context().Value(CacheControlKey).(*CacheControl); ok {
		return cacheControl.Enabled
	}

	// Check cache condition
	return c.cacheCondition(req)
}

// getCacheTTLForRequest gets the TTL for a request
func (c *Client) getCacheTTLForRequest(req *http.Request) time.Duration {
	// Check context-based cache control
	if cacheControl, ok := req.Context().Value(CacheControlKey).(*CacheControl); ok && cacheControl.TTL > 0 {
		return cacheControl.TTL
	}

	// Use default TTL
	return c.cacheTTL
}

// WithContextCacheEnabled creates a context that enables caching for the request
func WithContextCacheEnabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, CacheControlKey, &CacheControl{Enabled: true})
}

// WithContextCacheDisabled creates a context that disables caching for the request
func WithContextCacheDisabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, CacheControlKey, &CacheControl{Enabled: false})
}

// WithContextCacheTTL creates a context with custom TTL for the request
func WithContextCacheTTL(ctx context.Context, ttl time.Duration) context.Context {
	cacheControl := &CacheControl{Enabled: true, TTL: ttl}
	return context.WithValue(ctx, CacheControlKey, cacheControl)
}
