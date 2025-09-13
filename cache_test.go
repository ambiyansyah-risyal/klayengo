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

const testCacheURL = "https://example.com"
const cacheControlNotFoundMsg = "CacheControl not found in context"
const testData = "test data"
const testKey = "test-key"
const keyFormat = "key-%d"
const contentTypeHeader = "Content-Type"
const testResponse = "test response"
const writeResponseErrorMsg = "Failed to write response: %v"
const expectedFormatMsg = "Expected '%s', got '%s'"
const expectedContentTypeFormatMsg = "Expected Content-Type '%s', got '%s'"

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func (e *errorReader) Close() error {
	return nil
}

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

	_, found := cache.Get("nonexistent")
	if found {
		t.Error("Expected false for non-existent key")
	}

	entry := &CacheEntry{
		Body:       []byte(testData),
		StatusCode: 200,
		Header:     make(http.Header),
		ExpiresAt:  time.Now().Add(1 * time.Hour),
	}

	cache.Set(testKey, entry, 1*time.Hour)

	retrieved, found := cache.Get(testKey)
	if !found {
		t.Error("Expected true for existing key")
	}

	if string(retrieved.Body) != testData {
		t.Errorf(expectedFormatMsg, testData, string(retrieved.Body))
	}

	if retrieved.StatusCode != 200 {
		t.Errorf(expectedStatus200Msg, retrieved.StatusCode)
	}
}

func TestInMemoryCacheExpiration(t *testing.T) {
	cache := NewInMemoryCache()

	entry := &CacheEntry{
		Body:       []byte(testData),
		StatusCode: 200,
		Header:     make(http.Header),
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
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
		Body:       []byte(testData),
		StatusCode: 200,
		Header:     make(http.Header),
	}

	cache.Set(testKey, entry, 1*time.Hour)

	stored, exists := cache.Get(testKey)
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
		Body:       []byte(testData),
		StatusCode: 200,
		Header:     make(http.Header),
	}

	cache.Set(testKey, entry, 1*time.Hour)
	cache.Delete(testKey)

	_, exists := cache.Get(testKey)
	if exists {
		t.Error("Entry should have been deleted")
	}
}

func TestInMemoryCacheClear(t *testing.T) {
	cache := NewInMemoryCache()

	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Body:       []byte(testData),
			StatusCode: 200,
			Header:     make(http.Header),
		}
		cache.Set(fmt.Sprintf(keyFormat, i), entry, 1*time.Hour)
	}

	for i := 0; i < 5; i++ {
		_, exists := cache.Get(fmt.Sprintf(keyFormat, i))
		if !exists {
			t.Errorf("Entry %d should exist before clear", i)
		}
	}

	cache.Clear()

	for i := 0; i < 5; i++ {
		_, exists := cache.Get(fmt.Sprintf(keyFormat, i))
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
			contentTypeHeader: []string{contentTypeJSON},
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

	if resp.Header.Get(contentTypeHeader) != contentTypeJSON {
		t.Errorf(expectedContentTypeMsg, resp.Header.Get(contentTypeHeader))
	}
}

func TestCreateCacheEntry(t *testing.T) {
	client := New()

	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			contentTypeHeader: []string{contentTypeJSON},
			"Content-Length":  []string{"13"},
		},
		Body: io.NopCloser(strings.NewReader(testResponse)),
	}

	entry := client.createCacheEntry(resp)

	if entry.StatusCode != 200 {
		t.Errorf(expectedStatus200Msg, entry.StatusCode)
	}

	if string(entry.Body) != testResponse {
		t.Errorf(expectedFormatMsg, testResponse, string(entry.Body))
	}

	if entry.Header.Get(contentTypeHeader) != contentTypeJSON {
		t.Errorf(expectedContentTypeFormatMsg, contentTypeJSON, entry.Header.Get(contentTypeHeader))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read original response body: %v", err)
	}

	if string(body) != testResponse {
		t.Error("Original response body not properly restored")
	}
}

func TestCreateCacheEntryWithReadError(t *testing.T) {
	client := New()

	resp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       &errorReader{err: fmt.Errorf("read error")},
	}

	entry := client.createCacheEntry(resp)

	if entry != nil {
		t.Error("Expected nil entry when body read fails")
	}
}

func TestDefaultCacheKeyFunc(t *testing.T) {
	req, _ := http.NewRequest("GET", testCacheURL+"/api/data?id=123", nil)

	key := DefaultCacheKeyFunc(req)

	expected := "GET:" + testCacheURL + "/api/data?id=123"
	if key != expected {
		t.Errorf(expectedFormatMsg, expected, key)
	}
}

func TestDefaultCacheKeyFuncWithNilURL(t *testing.T) {
	req, _ := http.NewRequest("GET", "", nil)
	req.URL = nil

	key := DefaultCacheKeyFunc(req)

	expected := "GET:"
	if key != expected {
		t.Errorf(expectedFormatMsg, expected, key)
	}
}

func TestDefaultCacheCondition(t *testing.T) {
	getReq, _ := http.NewRequest("GET", testCacheURL+"/api/data", nil)
	postReq, _ := http.NewRequest("POST", testCacheURL+"/api/data", nil)

	if !DefaultCacheCondition(getReq) {
		t.Error("Expected GET request to be cacheable")
	}

	if DefaultCacheCondition(postReq) {
		t.Error("Expected POST request to not be cacheable")
	}
}

func TestShouldCacheRequestWithContextControl(t *testing.T) {
	client := New(WithCache(5 * time.Minute))

	ctx := WithContextCacheEnabled(context.Background())
	req := (&http.Request{}).WithContext(ctx)
	req.Method = "POST"
	req.URL, _ = req.URL.Parse(testCacheURL)

	if !client.shouldCacheRequest(req) {
		t.Error("Expected true when context enables caching")
	}

	ctx = WithContextCacheDisabled(context.Background())
	req = (&http.Request{}).WithContext(ctx)
	req.Method = "GET"
	req.URL, _ = req.URL.Parse(testCacheURL)

	if client.shouldCacheRequest(req) {
		t.Error("Expected false when context disables caching")
	}
}

func TestGetCacheTTLForRequest(t *testing.T) {
	client := New(WithCache(5 * time.Minute))

	req, _ := http.NewRequest("GET", testCacheURL, nil)

	ttl := client.getCacheTTLForRequest(req)
	if ttl != 5*time.Minute {
		t.Errorf("Expected TTL 5m, got %v", ttl)
	}

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
		t.Fatal(cacheControlNotFoundMsg)
	}

	if !cacheControl.Enabled {
		t.Error("Expected cache to be enabled")
	}
}

func TestWithContextCacheDisabled(t *testing.T) {
	ctx := WithContextCacheDisabled(context.Background())

	cacheControl, ok := ctx.Value(CacheControlKey).(*CacheControl)
	if !ok {
		t.Fatal(cacheControlNotFoundMsg)
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
		t.Fatal(cacheControlNotFoundMsg)
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
		w.Header().Set(contentTypeHeader, contentTypeJSON)
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(`{"data": "test"}`)); err != nil {
			t.Fatalf(writeResponseErrorMsg, err)
		}
	}))
	defer server.Close()

	client := New(WithCache(1 * time.Hour))

	resp1, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	_ = resp1.Body.Close()

	if callCount != 1 {
		t.Errorf("Expected 1 server call, got %d", callCount)
	}

	resp2, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	_ = resp2.Body.Close()

	if callCount != 1 {
		t.Errorf("Expected still 1 server call (cached), got %d", callCount)
	}

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

	customKeyFunc := func(req *http.Request) string {
		return req.Method + ":" + req.URL.Path
	}

	client := New(
		WithCache(1*time.Hour),
		WithCacheKeyFunc(customKeyFunc),
	)

	url1 := server.URL + "?param1=value1"
	url2 := server.URL + "?param2=value2"

	resp1, err := client.Get(context.Background(), url1)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}
	_ = resp1.Body.Close()

	resp2, err := client.Get(context.Background(), url2)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}
	_ = resp2.Body.Close()

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

	customCondition := func(req *http.Request) bool {
		return req.Method == "POST"
	}

	client := New(
		WithCache(1*time.Hour),
		WithCacheCondition(customCondition),
	)

	resp1, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	_ = resp1.Body.Close()

	resp2, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Second GET request failed: %v", err)
	}
	_ = resp2.Body.Close()

	if callCount != 2 {
		t.Errorf("Expected 2 server calls for GET, got %d", callCount)
	}

	resp3, err := client.Post(context.Background(), server.URL, contentTypeJSON, bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("POST request failed: %v", err)
	}
	_ = resp3.Body.Close()

	resp4, err := client.Post(context.Background(), server.URL, contentTypeJSON, bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatalf("Second POST request failed: %v", err)
	}
	_ = resp4.Body.Close()

	if callCount != 3 {
		t.Errorf("Expected 3 server calls total (POST cached), got %d", callCount)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := NewInMemoryCache()
	key := testKey
	entry := &CacheEntry{
		Response:   &http.Response{StatusCode: 200},
		Body:       []byte(testData),
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
		Body:       []byte(testData),
		StatusCode: 200,
		Header:     make(http.Header),
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf(keyFormat, i)
		cache.Set(key, entry, time.Hour)
	}
}

func BenchmarkCacheConcurrentAccess(b *testing.B) {
	cache := NewInMemoryCache()
	entry := &CacheEntry{
		Response:   &http.Response{StatusCode: 200},
		Body:       []byte(testData),
		StatusCode: 200,
		Header:     make(http.Header),
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf(keyFormat, i%1000)
			cache.Set(key, entry, time.Hour)
			cache.Get(key)
			i++
		}
	})
}
