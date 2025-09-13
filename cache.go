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

type InMemoryCache struct {
	shards    []*cacheShard
	numShards int
}

type cacheShard struct {
	mu    sync.RWMutex
	store map[string]*CacheEntry
}

func NewInMemoryCache() *InMemoryCache {
	numShards := 16
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

func (c *InMemoryCache) getShard(key string) *cacheShard {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	return c.shards[hash.Sum32()%uint32(c.numShards)]
}

func (c *InMemoryCache) Get(key string) (*CacheEntry, bool) {
	shard := c.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	entry, exists := shard.store[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(shard.store, key)
		return nil, false
	}

	return entry, true
}

func (c *InMemoryCache) Set(key string, entry *CacheEntry, ttl time.Duration) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	entry.ExpiresAt = time.Now().Add(ttl)
	shard.store[key] = entry
}

func (c *InMemoryCache) Delete(key string) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	delete(shard.store, key)
}

func (c *InMemoryCache) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.store = make(map[string]*CacheEntry)
		shard.mu.Unlock()
	}
}

func (c *Client) createResponseFromCache(entry *CacheEntry) *http.Response {
	resp := &http.Response{
		StatusCode: entry.StatusCode,
		Header:     entry.Header,
		Body:       io.NopCloser(bytes.NewReader(entry.Body)),
	}
	return resp
}

func (c *Client) createCacheEntry(resp *http.Response) *CacheEntry {
	const maxCacheSize = 10 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCacheSize))
	if err != nil && err != io.EOF {
		return nil
	}

	_ = resp.Body.Close()

	resp.Body = io.NopCloser(bytes.NewReader(body))

	return &CacheEntry{
		Response:   resp,
		Body:       body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
	}
}

func DefaultCacheKeyFunc(req *http.Request) string {
	if req.URL == nil {
		return req.Method + ":"
	}

	var buf []byte
	buf = append(buf, req.Method...)
	buf = append(buf, ':')
	buf = append(buf, req.URL.String()...)

	return string(buf)
}

func DefaultCacheCondition(req *http.Request) bool {
	return req.Method == "GET"
}

func (c *Client) shouldCacheRequest(req *http.Request) bool {
	if c.cache == nil {
		return false
	}

	if cacheControl, ok := req.Context().Value(CacheControlKey).(*CacheControl); ok {
		return cacheControl.Enabled
	}

	return c.cacheCondition(req)
}

func (c *Client) getCacheTTLForRequest(req *http.Request) time.Duration {
	if cacheControl, ok := req.Context().Value(CacheControlKey).(*CacheControl); ok && cacheControl.TTL > 0 {
		return cacheControl.TTL
	}

	return c.cacheTTL
}

func WithContextCacheEnabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, CacheControlKey, &CacheControl{Enabled: true})
}

func WithContextCacheDisabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, CacheControlKey, &CacheControl{Enabled: false})
}

func WithContextCacheTTL(ctx context.Context, ttl time.Duration) context.Context {
	cacheControl := &CacheControl{Enabled: true, TTL: ttl}
	return context.WithValue(ctx, CacheControlKey, cacheControl)
}
