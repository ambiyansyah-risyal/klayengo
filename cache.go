package klayengo

import (
	"bytes"
	"context"
	"hash/fnv"
	"io"
	"net/http"
	"sync"
	"time"
)

// InMemoryCache is a sharded in-memory cache implementation for better concurrency
type InMemoryCache struct {
	shards    []*cacheShard
	numShards int
}

// cacheShard represents a single shard of the cache
type cacheShard struct {
	mu    sync.RWMutex
	store map[string]*CacheEntry
}

// NewInMemoryCache creates a new sharded in-memory cache
func NewInMemoryCache() *InMemoryCache {
	numShards := 16 // Use 16 shards for good concurrency
	shards := make([]*cacheShard, numShards)
	for i := range shards {
		shards[i] = &cacheShard{
			store: make(map[string]*CacheEntry),
		}
	}
	return &InMemoryCache{
		shards:    shards,
		numShards: numShards,
	}
}

// getShard returns the shard for a given key
func (c *InMemoryCache) getShard(key string) *cacheShard {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	return c.shards[hash.Sum32()%uint32(c.numShards)]
}

// Get retrieves a cached entry
func (c *InMemoryCache) Get(key string) (*CacheEntry, bool) {
	shard := c.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	entry, exists := shard.store[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, remove it
		delete(shard.store, key)
		return nil, false
	}

	return entry, true
}

// Set stores a cache entry
func (c *InMemoryCache) Set(key string, entry *CacheEntry, ttl time.Duration) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	entry.ExpiresAt = time.Now().Add(ttl)
	shard.store[key] = entry
}

// Delete removes a cache entry
func (c *InMemoryCache) Delete(key string) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	delete(shard.store, key)
}

// Clear removes all cache entries
func (c *InMemoryCache) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.store = make(map[string]*CacheEntry)
		shard.mu.Unlock()
	}
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

// createCacheEntry creates a cache entry from an HTTP response with size limits
func (c *Client) createCacheEntry(resp *http.Response) *CacheEntry {
	// Limit cache entry size to prevent memory issues (10MB default)
	const maxCacheSize = 10 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCacheSize))
	if err != nil && err != io.EOF {
		// If we can't read the body, don't cache
		return nil
	}

	resp.Body.Close()

	// Restore the body for the caller
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return &CacheEntry{
		Response:   resp,
		Body:       body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(), // Clone to avoid sharing
	}
}

// DefaultCacheKeyFunc generates a cache key from the request using efficient string building
func DefaultCacheKeyFunc(req *http.Request) string {
	if req.URL == nil {
		return req.Method + ":"
	}

	// Pre-allocate buffer for efficiency
	var buf []byte
	buf = append(buf, req.Method...)
	buf = append(buf, ':')
	buf = append(buf, req.URL.String()...)

	return string(buf)
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
