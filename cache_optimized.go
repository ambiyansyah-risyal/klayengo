package klayengo

import (
	"context"
	"hash/fnv"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Optimal shard count based on typical CPU core counts
	DefaultShardCount = 64
	// Cache line size for optimal memory layout
	CacheLineSize = 64
	// LRU eviction threshold (when to start evicting)
	EvictionThreshold = 0.85
)

// OptimizedCache implements a high-performance sharded cache with
// reduced lock contention, hot data promotion, and atomic access counters.
type OptimizedCache struct {
	shards    []*optimizedCacheShard
	numShards int64
	shardMask int64 // optimization for modulo operation

	// Global statistics - lock-free counters
	totalHits      int64
	totalMisses    int64
	totalSets      int64
	totalEvictions int64
}

// optimizedCacheShard represents a single cache shard with optimized locking
type optimizedCacheShard struct {
	// Cache line aligned to prevent false sharing
	_pad0 [CacheLineSize]byte //nolint:unused // padding for cache line alignment

	// Hot path data - grouped for cache locality
	store map[string]*optimizedCacheEntry
	mu    sync.RWMutex

	// LRU tracking for efficient eviction
	head, tail *optimizedCacheEntry

	// Shard-level statistics
	hits      int64
	misses    int64
	evictions int64
	size      int64
	maxSize   int64

	_pad1 [CacheLineSize]byte //nolint:unused // padding for cache line alignment
}

// optimizedCacheEntry with LRU tracking and atomic access counters
type optimizedCacheEntry struct {
	// Core cache data
	key       string
	response  *http.Response
	body      []byte
	header    http.Header
	expiresAt int64 // nanoseconds for faster comparison

	// HTTP cache semantics
	etag         string
	lastModified *time.Time
	maxAge       *time.Duration
	staleAt      int64 // nanoseconds
	isStale      int32 // atomic boolean

	// LRU chain pointers
	prev, next *optimizedCacheEntry

	// Performance tracking - atomic counters
	accessCount int64
	lastAccess  int64 // nanoseconds

	// Hot data promotion
	hotPromoted int32 // atomic boolean
}

// NewOptimizedCache creates a high-performance sharded cache
func NewOptimizedCache() *OptimizedCache {
	return NewOptimizedCacheWithSize(DefaultShardCount, 1000) // 1000 entries per shard default
}

// NewOptimizedCacheWithSize creates a cache with specified shard count and size per shard
func NewOptimizedCacheWithSize(shardCount, maxSizePerShard int) *OptimizedCache {
	if shardCount <= 0 {
		shardCount = runtime.NumCPU() * 2 // 2 shards per CPU core
	}
	if maxSizePerShard <= 0 {
		maxSizePerShard = 1000
	}

	// Ensure shard count is power of 2 for efficient masking
	shardCount = nextPowerOf2(shardCount)

	shards := make([]*optimizedCacheShard, shardCount)
	for i := range shards {
		shards[i] = &optimizedCacheShard{
			store:   make(map[string]*optimizedCacheEntry),
			maxSize: int64(maxSizePerShard),
		}
	}

	return &OptimizedCache{
		shards:    shards,
		numShards: int64(shardCount),
		shardMask: int64(shardCount - 1),
	}
}

// Get retrieves an entry with hot path optimization and LRU promotion
func (c *OptimizedCache) Get(key string) (*CacheEntry, bool) {
	shard := c.getShard(key)

	// Fast path: try read lock first
	shard.mu.RLock()
	entry := shard.store[key]
	if entry == nil {
		shard.mu.RUnlock()
		atomic.AddInt64(&shard.misses, 1)
		atomic.AddInt64(&c.totalMisses, 1)
		return nil, false
	}

	// Quick expiration check
	now := time.Now().UnixNano()
	if now > entry.expiresAt {
		shard.mu.RUnlock()
		// Expired entry - clean up with write lock
		shard.mu.Lock()
		delete(shard.store, key)
		c.removeLRU(shard, entry)
		shard.mu.Unlock()

		atomic.AddInt64(&shard.misses, 1)
		atomic.AddInt64(&c.totalMisses, 1)
		return nil, false
	}

	// Update access statistics atomically
	atomic.AddInt64(&entry.accessCount, 1)
	atomic.StoreInt64(&entry.lastAccess, now)

	// Create response copy while holding read lock - minimize allocations
	statusCode := 200 // default
	if entry.response != nil {
		statusCode = entry.response.StatusCode
	}

	// Pre-allocate to avoid multiple allocations
	cacheEntry := &CacheEntry{
		Body:         entry.body, // Share byte slice (read-only)
		StatusCode:   statusCode,
		Header:       entry.header, // Share header (caller should not modify)
		ExpiresAt:    time.Unix(0, entry.expiresAt),
		ETag:         entry.etag,
		LastModified: entry.lastModified,
		MaxAge:       entry.maxAge,
		IsStale:      atomic.LoadInt32(&entry.isStale) == 1,
	}

	shard.mu.RUnlock()

	// Hot data promotion - move frequently accessed items to front
	if atomic.LoadInt64(&entry.accessCount)%10 == 0 { // promote every 10th access
		c.promoteHotEntry(shard, entry)
	}

	atomic.AddInt64(&shard.hits, 1)
	atomic.AddInt64(&c.totalHits, 1)
	return cacheEntry, true
}

// Set stores an entry with LRU eviction and automatic cleanup
func (c *OptimizedCache) Set(key string, entry *CacheEntry, ttl time.Duration) {
	shard := c.getShard(key)
	now := time.Now()

	optimizedEntry := &optimizedCacheEntry{
		key:          key,
		body:         entry.Body,
		header:       entry.Header.Clone(),
		expiresAt:    now.Add(ttl).UnixNano(),
		etag:         entry.ETag,
		lastModified: entry.LastModified,
		maxAge:       entry.MaxAge,
		lastAccess:   now.UnixNano(),
		accessCount:  1,
		response:     entry.Response, // Store the response
	}

	// If no response provided, create a minimal one for StatusCode
	if optimizedEntry.response == nil {
		optimizedEntry.response = &http.Response{
			StatusCode: entry.StatusCode,
			Header:     entry.Header,
		}
	}

	if entry.StaleAt != nil {
		optimizedEntry.staleAt = entry.StaleAt.UnixNano()
	}

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Check if we need to evict old entries
	if shard.size >= int64(float64(shard.maxSize)*EvictionThreshold) {
		c.evictLRU(shard)
	}

	// Remove existing entry if present
	if existing := shard.store[key]; existing != nil {
		c.removeLRU(shard, existing)
		shard.size--
	}

	// Add new entry
	shard.store[key] = optimizedEntry
	c.addToLRU(shard, optimizedEntry)
	shard.size++

	atomic.AddInt64(&c.totalSets, 1)
}

// Delete removes an entry from the cache
func (c *OptimizedCache) Delete(key string) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if entry := shard.store[key]; entry != nil {
		delete(shard.store, key)
		c.removeLRU(shard, entry)
		shard.size--
	}
}

// Clear removes all entries from all shards
func (c *OptimizedCache) Clear() {
	for _, shard := range c.shards {
		shard.mu.Lock()
		shard.store = make(map[string]*optimizedCacheEntry)
		shard.head = nil
		shard.tail = nil
		shard.size = 0
		shard.mu.Unlock()
	}

	// Reset global counters
	atomic.StoreInt64(&c.totalHits, 0)
	atomic.StoreInt64(&c.totalMisses, 0)
	atomic.StoreInt64(&c.totalSets, 0)
	atomic.StoreInt64(&c.totalEvictions, 0)
}

// getShard returns the shard for a given key using optimized hashing
func (c *OptimizedCache) getShard(key string) *optimizedCacheShard {
	hash := fnv.New64a()
	// Use byte slice conversion for better compatibility
	hash.Write([]byte(key))
	return c.shards[hash.Sum64()&uint64(c.shardMask)]
}

// LRU management functions

func (c *OptimizedCache) addToLRU(shard *optimizedCacheShard, entry *optimizedCacheEntry) {
	if shard.head == nil {
		shard.head = entry
		shard.tail = entry
		return
	}

	entry.next = shard.head
	shard.head.prev = entry
	shard.head = entry
}

func (c *OptimizedCache) removeLRU(shard *optimizedCacheShard, entry *optimizedCacheEntry) {
	if entry.prev != nil {
		entry.prev.next = entry.next
	} else {
		shard.head = entry.next
	}

	if entry.next != nil {
		entry.next.prev = entry.prev
	} else {
		shard.tail = entry.prev
	}

	entry.prev = nil
	entry.next = nil
}

func (c *OptimizedCache) promoteHotEntry(shard *optimizedCacheShard, entry *optimizedCacheEntry) {
	if atomic.CompareAndSwapInt32(&entry.hotPromoted, 0, 1) {
		shard.mu.Lock()
		c.removeLRU(shard, entry)
		c.addToLRU(shard, entry)
		shard.mu.Unlock()
		atomic.StoreInt32(&entry.hotPromoted, 0) // reset for next promotion
	}
}

func (c *OptimizedCache) evictLRU(shard *optimizedCacheShard) {
	// Evict from tail (oldest)
	if shard.tail == nil {
		return
	}

	evicted := shard.tail
	delete(shard.store, evicted.key)
	c.removeLRU(shard, evicted)
	shard.size--

	atomic.AddInt64(&shard.evictions, 1)
	atomic.AddInt64(&c.totalEvictions, 1)
}

// GetStats returns cache performance statistics
func (c *OptimizedCache) GetStats() CacheStats {
	totalSize := int64(0)
	totalCapacity := int64(0)
	shardStats := make([]ShardStats, len(c.shards))

	for i, shard := range c.shards {
		shard.mu.RLock()
		size := shard.size
		capacity := shard.maxSize
		hits := atomic.LoadInt64(&shard.hits)
		misses := atomic.LoadInt64(&shard.misses)
		evictions := atomic.LoadInt64(&shard.evictions)
		shard.mu.RUnlock()

		totalSize += size
		totalCapacity += capacity

		shardStats[i] = ShardStats{
			Size:      size,
			Capacity:  capacity,
			Hits:      hits,
			Misses:    misses,
			Evictions: evictions,
		}
	}

	totalHits := atomic.LoadInt64(&c.totalHits)
	totalMisses := atomic.LoadInt64(&c.totalMisses)
	hitRatio := float64(0)
	if totalHits+totalMisses > 0 {
		hitRatio = float64(totalHits) / float64(totalHits+totalMisses)
	}

	return CacheStats{
		TotalSize:      totalSize,
		TotalCapacity:  totalCapacity,
		TotalHits:      totalHits,
		TotalMisses:    totalMisses,
		TotalSets:      atomic.LoadInt64(&c.totalSets),
		TotalEvictions: atomic.LoadInt64(&c.totalEvictions),
		HitRatio:       hitRatio,
		ShardCount:     int64(len(c.shards)),
		ShardStats:     shardStats,
	}
}

// Supporting types for statistics

type CacheStats struct {
	TotalSize      int64
	TotalCapacity  int64
	TotalHits      int64
	TotalMisses    int64
	TotalSets      int64
	TotalEvictions int64
	HitRatio       float64
	ShardCount     int64
	ShardStats     []ShardStats
}

type ShardStats struct {
	Size      int64
	Capacity  int64
	Hits      int64
	Misses    int64
	Evictions int64
}

// Utility functions

func nextPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// Implement CacheProvider interface for backwards compatibility

type OptimizedCacheProvider struct {
	cache      *OptimizedCache
	mode       CacheMode
	defaultTTL time.Duration
}

func NewOptimizedCacheProvider(cache *OptimizedCache, defaultTTL time.Duration, mode CacheMode) *OptimizedCacheProvider {
	return &OptimizedCacheProvider{
		cache:      cache,
		mode:       mode,
		defaultTTL: defaultTTL,
	}
}

func (p *OptimizedCacheProvider) Get(ctx context.Context, key string) (*http.Response, bool) {
	entry, ok := p.cache.Get(key)
	if !ok {
		return nil, false
	}

	// Convert back to http.Response (this could be optimized further)
	resp := &http.Response{
		StatusCode: entry.StatusCode,
		Header:     entry.Header,
		Body:       nil, // Body will be set by caller
	}

	return resp, true
}

func (p *OptimizedCacheProvider) Set(ctx context.Context, key string, resp *http.Response, ttl time.Duration) {
	if ttl <= 0 {
		ttl = p.defaultTTL
	}

	// Create cache entry from response (this could be optimized further)
	entry := &CacheEntry{
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(),
	}

	p.cache.Set(key, entry, ttl)
}

func (p *OptimizedCacheProvider) Invalidate(ctx context.Context, key string) {
	p.cache.Delete(key)
}
