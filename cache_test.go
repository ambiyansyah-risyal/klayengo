package klayengo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const writeResponseErrorMsg = "Failed to write response: %v"

func TestNewInMemoryCache(t *testing.T) {
	cache := NewInMemoryCache()

	if cache == nil {
		t.Fatal("NewInMemoryCache() returned nil")
	}

	if cache.shards == nil {
		t.Error("Cache shards not initialized")
	}

	if len(cache.shards) != cache.numShards {
		t.Errorf("Expected %d shards, got %d", cache.numShards, len(cache.shards))
	}
}

func TestInMemoryCacheGet(t *testing.T) {
	cache := NewInMemoryCache()

	// Test getting non-existent key
	_, found := cache.Get("nonexistent")
	if found {
		t.Error("Expected false for non-existent key")
	}

	// Set and get a cache entry
	entry := &CacheEntry{
		Body:       []byte("test data"),
		StatusCode: 200,
		Header:     make(http.Header),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	cache.Set("test-key", entry, 1*time.Hour)

	retrieved, found := cache.Get("test-key")
	if !found {
		t.Error("Expected true for existing key")
	}

	if string(retrieved.Body) != "test data" {
		t.Errorf("Expected 'test data', got '%s'", string(retrieved.Body))
	}

	if retrieved.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", retrieved.StatusCode)
	}
}

func TestInMemoryCacheExpiration(t *testing.T) {
	cache := NewInMemoryCache()

	entry := &CacheEntry{
		Body:       []byte("test data"),
		StatusCode: 200,
		Header:     make(http.Header),
		ExpiresAt:  time.Now().Add(-1 * time.Hour), // Already expired
	}

	cache.Set("expired-key", entry, -1*time.Hour)

	_, found := cache.Get("expired-key")
	if found {
		t.Error("Expected expired entry to not be found")
	}
}

func TestInMemoryCacheSet(t *testing.T) {
	cache := NewInMemoryCache()

	entry := &CacheEntry{
		Body:       []byte("test data"),
		StatusCode: 200,
		Header:     make(http.Header),
	}

	cache.Set("test-key", entry, 1*time.Hour)

	// Check if entry exists by trying to get it
	stored, exists := cache.Get("test-key")
	if !exists {
		t.Error("Entry not stored in cache")
	}

	if stored.ExpiresAt.Before(time.Now()) {
		t.Error("Entry expiration time not set correctly")
	}
}

func TestInMemoryCacheDelete(t *testing.T) {
	cache := NewInMemoryCache()

	entry := &CacheEntry{
		Body:       []byte("test data"),
		StatusCode: 200,
		Header:     make(http.Header),
	}

	cache.Set("test-key", entry, 1*time.Hour)
	cache.Delete("test-key")

	// Check if entry was deleted
	_, exists := cache.Get("test-key")
	if exists {
		t.Error("Entry should have been deleted")
	}
}

func TestInMemoryCacheClear(t *testing.T) {
	cache := NewInMemoryCache()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Body:       []byte("test data"),
			StatusCode: 200,
			Header:     make(http.Header),
		}
		cache.Set(fmt.Sprintf("key-%d", i), entry, 1*time.Hour)
	}

	// Verify entries exist
	for i := 0; i < 5; i++ {
		_, exists := cache.Get(fmt.Sprintf("key-%d", i))
		if !exists {
			t.Errorf("Entry %d should exist before clear", i)
		}
	}

	cache.Clear()

	// Verify all entries are cleared
	for i := 0; i < 5; i++ {
		_, exists := cache.Get(fmt.Sprintf("key-%d", i))
		if exists {
			t.Errorf("Entry %d should not exist after clear", i)
		}
	}
}

func TestCreateResponseFromCache(t *testing.T) {
	client := New()

	entry := &CacheEntry{
		Body:       []byte("cached response"),
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	resp := client.createResponseFromCache(entry)

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != "cached response" {
		t.Errorf("Expected 'cached response', got '%s'", string(body))
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", resp.Header.Get("Content-Type"))
	}
}

func TestCreateCacheEntry(t *testing.T) {
	client := New()

	// Create a mock response
	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Content-Length": []string{"13"},
		},
		Body: io.NopCloser(strings.NewReader("test response")),
	}

	entry := client.createCacheEntry(resp)

	if entry.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", entry.StatusCode)
	}

	if string(entry.Body) != "test response" {
		t.Errorf("Expected 'test response', got '%s'", string(entry.Body))
	}

	if entry.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", entry.Header.Get("Content-Type"))
	}

	// Verify original response body is restored
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read original response body: %v", err)
	}

	if string(body) != "test response" {
		t.Error("Original response body not properly restored")
	}
}

func TestDefaultCacheKeyFunc(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/api/data?id=123", nil)

	key := DefaultCacheKeyFunc(req)

	expected := "GET:https://example.com/api/data?id=123"
	if key != expected {
		t.Errorf("Expected '%s', got '%s'", expected, key)
	}
}

func TestDefaultCacheCondition(t *testing.T) {
	getReq, _ := http.NewRequest("GET", "https://example.com/api/data", nil)
	postReq, _ := http.NewRequest("POST", "https://example.com/api/data", nil)

	if !DefaultCacheCondition(getReq) {
		t.Error("Expected GET request to be cacheable")
	}

	if DefaultCacheCondition(postReq) {
		t.Error("Expected POST request to not be cacheable")
	}
}

func TestShouldCacheRequest(t *testing.T) {
	client := New()

	// Test without cache
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	if client.shouldCacheRequest(req) {
		t.Error("Expected false when no cache is configured")
	}

	// Test with cache
	client = New(WithCache(5 * time.Minute))
	req, _ = http.NewRequest("GET", "https://example.com", nil)
	if !client.shouldCacheRequest(req) {
		t.Error("Expected true when cache is configured and condition met")
	}

	// Test with cache but POST request
	req, _ = http.NewRequest("POST", "https://example.com", nil)
	if client.shouldCacheRequest(req) {
		t.Error("Expected false for POST request")
	}
}

func TestGetCacheTTLForRequest(t *testing.T) {
	client := New(WithCache(5 * time.Minute))

	req, _ := http.NewRequest("GET", "https://example.com", nil)

	ttl := client.getCacheTTLForRequest(req)
	if ttl != 5*time.Minute {
		t.Errorf("Expected TTL 5m, got %v", ttl)
	}

	// Test with context override
	ctx := WithContextCacheTTL(context.Background(), 10*time.Minute)
	req = req.WithContext(ctx)

	ttl = client.getCacheTTLForRequest(req)
	if ttl != 10*time.Minute {
		t.Errorf("Expected TTL 10m, got %v", ttl)
	}
}

func TestWithContextCacheEnabled(t *testing.T) {
	ctx := WithContextCacheEnabled(context.Background())

	cacheControl, ok := ctx.Value(CacheControlKey).(*CacheControl)
	if !ok {
		t.Fatal("CacheControl not found in context")
	}

	if !cacheControl.Enabled {
		t.Error("Expected cache to be enabled")
	}
}

func TestWithContextCacheDisabled(t *testing.T) {
	ctx := WithContextCacheDisabled(context.Background())

	cacheControl, ok := ctx.Value(CacheControlKey).(*CacheControl)
	if !ok {
		t.Fatal("CacheControl not found in context")
	}

	if cacheControl.Enabled {
		t.Error("Expected cache to be disabled")
	}
}

func TestWithContextCacheTTL(t *testing.T) {
	customTTL := 30 * time.Minute
	ctx := WithContextCacheTTL(context.Background(), customTTL)

	cacheControl, ok := ctx.Value(CacheControlKey).(*CacheControl)
	if !ok {
		t.Fatal("CacheControl not found in context")
	}

	if !cacheControl.Enabled {
		t.Error("Expected cache to be enabled")
	}

	if cacheControl.TTL != customTTL {
		t.Errorf("Expected TTL %v, got %v", customTTL, cacheControl.TTL)
	}
}

func TestCachingInDo(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(`{"data": "test"}`)); err != nil {
			t.Fatalf(writeResponseErrorMsg, err)
		}
	}))
	defer server.Close()

	client := New(WithCache(1 * time.Hour))

	// First request should hit the server
	resp1, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	resp1.Body.Close()

	if callCount != 1 {
		t.Errorf("Expected 1 server call, got %d", callCount)
	}

	// Second request should use cache
	resp2, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	resp2.Body.Close()

	if callCount != 1 {
		t.Errorf("Expected still 1 server call (cached), got %d", callCount)
	}

	// Verify cached response content
	body, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("Failed to read cached response: %v", err)
	}

	if string(body) != `{"data": "test"}` {
		t.Errorf("Cached response content mismatch: %s", string(body))
	}
}

func TestCacheWithCustomKeyFunc(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("response")); err != nil {
			t.Fatalf(writeResponseErrorMsg, err)
		}
	}))
	defer server.Close()

	// Custom key function that ignores query parameters
	customKeyFunc := func(req *http.Request) string {
		return req.Method + ":" + req.URL.Path
	}

	client := New(
		WithCache(1*time.Hour),
		WithCacheKeyFunc(customKeyFunc),
	)

	// Make requests with different query parameters but same path
	url1 := server.URL + "?param1=value1"
	url2 := server.URL + "?param2=value2"

	resp1, err := client.Get(context.Background(), url1)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	resp1.Body.Close()

	resp2, err := client.Get(context.Background(), url2)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	resp2.Body.Close()

	// Should only call server once due to same cache key
	if callCount != 1 {
		t.Errorf("Expected 1 server call (cached), got %d", callCount)
	}
}

func TestCacheWithCustomCondition(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("response")); err != nil {
			t.Fatalf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Custom condition that caches POST requests
	customCondition := func(req *http.Request) bool {
		return req.Method == "POST"
	}

	client := New(
		WithCache(1*time.Hour),
		WithCacheCondition(customCondition),
	)

	// GET request should not be cached
	resp1, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	resp1.Body.Close()

	resp2, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Second GET request failed: %v", err)
	}
	resp2.Body.Close()

	// Should call server twice for GET requests
	if callCount != 2 {
		t.Errorf("Expected 2 server calls for GET, got %d", callCount)
	}

	// POST request should be cached
	resp3, err := client.Post(context.Background(), server.URL, "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	resp3.Body.Close()

	resp4, err := client.Post(context.Background(), server.URL, "application/json", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("Second POST request failed: %v", err)
	}
	resp4.Body.Close()

	// Should call server once more for POST (cached)
	if callCount != 3 {
		t.Errorf("Expected 3 server calls total (POST cached), got %d", callCount)
	}
}

// Benchmark tests for cache performance

func BenchmarkCacheGet(b *testing.B) {
	cache := NewInMemoryCache()
	key := "test-key"
	entry := &CacheEntry{
		Response:   &http.Response{StatusCode: 200},
		Body:       []byte("test data"),
		StatusCode: 200,
		Header:     make(http.Header),
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	cache.Set(key, entry, time.Hour)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(key)
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := NewInMemoryCache()
	entry := &CacheEntry{
		Response:   &http.Response{StatusCode: 200},
		Body:       []byte("test data"),
		StatusCode: 200,
		Header:     make(http.Header),
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i)
		cache.Set(key, entry, time.Hour)
	}
}

func BenchmarkCacheConcurrentAccess(b *testing.B) {
	cache := NewInMemoryCache()
	entry := &CacheEntry{
		Response:   &http.Response{StatusCode: 200},
		Body:       []byte("test data"),
		StatusCode: 200,
		Header:     make(http.Header),
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i%1000)
			cache.Set(key, entry, time.Hour)
			cache.Get(key)
			i++
		}
	})
}
