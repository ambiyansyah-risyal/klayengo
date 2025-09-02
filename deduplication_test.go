package klayengo

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const deduplicationTestURL = "http://example.com/test"

func TestDeduplicationTracker(t *testing.T) {
	tracker := NewDeduplicationTracker()

	key := "test-key"
	_, isOwner := tracker.GetOrCreateEntry(key)

	// First call should be the owner
	if !isOwner {
		t.Error("First call should be the owner")
	}

	// Owner should not wait - it should make the request
	// Simulate making the request...

	// Complete the entry
	testResp := &http.Response{StatusCode: 200}
	testErr := error(nil)
	tracker.Complete(key, testResp, testErr)

	// Second call should not be the owner
	entry2, isOwner2 := tracker.GetOrCreateEntry(key)
	if isOwner2 {
		t.Error("Second call should not be the owner")
	}

	// Second caller should wait and get the result
	resp2, err2 := entry2.Wait(context.Background())
	if resp2 != testResp || err2 != testErr {
		t.Errorf("Second waiter should receive result, got resp=%v, err=%v", resp2, err2)
	}
}

func TestDeduplicationIntegration(t *testing.T) {
	// Create a test server that simulates slow responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(200)
		if _, err := w.Write([]byte("OK")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	// Create client with deduplication enabled
	client := New(
		WithDeduplication(),
		WithMaxRetries(0), // Disable retries for this test
	)

	var wg sync.WaitGroup
	var responses []*http.Response
	var errors []error
	var mu sync.Mutex

	// Make 5 concurrent requests to the same endpoint
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", server.URL+"/test", nil)
			resp, err := client.Do(req)

			mu.Lock()
			responses = append(responses, resp)
			errors = append(errors, err)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// All requests should succeed
	for i, err := range errors {
		if err != nil {
			t.Errorf("Request %d failed: %v", i, err)
		}
	}

	// All responses should be valid
	for i, resp := range responses {
		if resp == nil {
			t.Errorf("Request %d got nil response", i)
			continue
		}
		if resp.StatusCode != 200 {
			t.Errorf("Request %d got status %d, expected 200", i, resp.StatusCode)
		}
	}
}

func TestDefaultDeduplicationKeyFunc(t *testing.T) {
	req1, _ := http.NewRequest("GET", deduplicationTestURL, nil)
	req2, _ := http.NewRequest("GET", deduplicationTestURL, nil)
	req3, _ := http.NewRequest("POST", deduplicationTestURL, nil)

	key1 := DefaultDeduplicationKeyFunc(req1)
	key2 := DefaultDeduplicationKeyFunc(req2)
	key3 := DefaultDeduplicationKeyFunc(req3)

	// Same method and URL should have same key
	if key1 != key2 {
		t.Errorf("Same requests should have same key: %s != %s", key1, key2)
	}

	// Different method should have different key
	if key1 == key3 {
		t.Errorf("Different methods should have different keys: %s == %s", key1, key3)
	}

	// Key should be non-empty
	if key1 == "" {
		t.Error("Key should not be empty")
	}
}

func TestDefaultDeduplicationKeyFuncWithBody(t *testing.T) {
	bodyContent := "test body content"

	// Test POST with body
	req1, _ := http.NewRequest("POST", deduplicationTestURL, strings.NewReader(bodyContent))
	req1.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(bodyContent)), nil
	}

	// Test POST with same body
	req2, _ := http.NewRequest("POST", deduplicationTestURL, strings.NewReader(bodyContent))
	req2.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(bodyContent)), nil
	}

	// Test POST with different body
	req3, _ := http.NewRequest("POST", deduplicationTestURL, strings.NewReader("different body"))
	req3.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader("different body")), nil
	}

	key1 := DefaultDeduplicationKeyFunc(req1)
	key2 := DefaultDeduplicationKeyFunc(req2)
	key3 := DefaultDeduplicationKeyFunc(req3)

	// Same body should have same key
	if key1 != key2 {
		t.Errorf("Same POST body should have same key: %s != %s", key1, key2)
	}

	// Different body should have different key
	if key1 == key3 {
		t.Errorf("Different POST body should have different keys: %s == %s", key1, key3)
	}
}

func TestDefaultDeduplicationKeyFuncWithBodyError(t *testing.T) {
	// Test POST with body but GetBody returns error
	req, _ := http.NewRequest("POST", deduplicationTestURL, strings.NewReader("test"))
	req.GetBody = func() (io.ReadCloser, error) {
		return nil, fmt.Errorf("body read error")
	}

	key := DefaultDeduplicationKeyFunc(req)

	// Should still generate a key even with body read error
	if key == "" {
		t.Error("Key should not be empty even with body read error")
	}
}

func TestDeduplicationCondition(t *testing.T) {
	getReq, _ := http.NewRequest("GET", deduplicationTestURL, nil)
	postReq, _ := http.NewRequest("POST", deduplicationTestURL, nil)
	putReq, _ := http.NewRequest("PUT", deduplicationTestURL, nil)
	deleteReq, _ := http.NewRequest("DELETE", deduplicationTestURL, nil)
	headReq, _ := http.NewRequest("HEAD", deduplicationTestURL, nil)
	optionsReq, _ := http.NewRequest("OPTIONS", deduplicationTestURL, nil)

	tests := []struct {
		req      *http.Request
		expected bool
	}{
		{getReq, true},
		{postReq, false},
		{putReq, false},
		{deleteReq, false},
		{headReq, true},
		{optionsReq, true},
	}

	for _, test := range tests {
		result := DefaultDeduplicationCondition(test.req)
		if result != test.expected {
			t.Errorf("Method %s: expected %v, got %v", test.req.Method, test.expected, result)
		}
	}
}

// Benchmark tests for deduplication performance

func BenchmarkDefaultDeduplicationKeyFunc(b *testing.B) {
	req, _ := http.NewRequest("GET", "https://api.example.com/users/123?param=value", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultDeduplicationKeyFunc(req)
	}
}

func BenchmarkDefaultDeduplicationKeyFuncWithBody(b *testing.B) {
	body := strings.NewReader(`{"name": "test", "value": 123}`)
	req, _ := http.NewRequest("POST", "https://api.example.com/users", body)
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(`{"name": "test", "value": 123}`)), nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultDeduplicationKeyFunc(req)
	}
}

func BenchmarkDeduplicationTracker(b *testing.B) {
	tracker := NewDeduplicationTracker()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key-%d", i%1000)
		entry, _ := tracker.GetOrCreateEntry(key)
		_ = entry
	}
}

func TestDeduplicationPerformanceBenefit(t *testing.T) {
	// This test demonstrates the performance benefit of deduplication
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		time.Sleep(10 * time.Millisecond) // Simulate network latency
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("response")); err != nil {
			t.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server.Close()

	client := New(WithDeduplication())

	// Make multiple concurrent requests to the same endpoint
	const numRequests = 10
	var wg sync.WaitGroup
	start := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(context.Background(), server.URL)
			if err != nil {
				t.Errorf("Request failed: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	// Only one actual network request should be made
	if callCount != 1 {
		t.Errorf("Expected 1 network call, got %d", callCount)
	}

	// All requests should complete quickly due to deduplication
	maxExpectedDuration := 50 * time.Millisecond // Much less than 10ms * 10 = 100ms
	if duration > maxExpectedDuration {
		t.Errorf("Requests took too long: %v (expected < %v)", duration, maxExpectedDuration)
	}

	t.Logf("✅ Deduplication benefit: %d concurrent requests completed in %v with only %d network call",
		numRequests, duration, callCount)
}
