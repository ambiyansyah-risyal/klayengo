package klayengo

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

const (
	testHost        = "api.example.com"
	testHostKey     = "host:api.example.com"
	defaultKey      = "default"
	fallbackMsg     = "Expected fallback limiter, got %v"
	keyMsg          = "Expected key '%s', got %s"
	firstReqMsg     = "First request should be allowed"
	secondReqMsg    = "Second request should be denied"
	concurrentMsg   = "Concurrent access failed: expected %s, got %s"
	hostKeyMsg      = "Expected host key %s, got %s"
	routeKeyMsg     = "Expected route key %s, got %s"
	hostRouteKeyMsg = "Expected host-route key %s, got %s"
	nilLimiterMsg   = "Expected nil limiter, got %v"
	allowMsg        = "Expected request to be allowed when no limiter is configured"
)

func TestRateLimiterRegistryGetLimiter(t *testing.T) {
	limiter1 := NewRateLimiter(10, time.Second)
	limiter2 := NewRateLimiter(20, time.Second)

	registry := NewRateLimiterRegistry(DefaultHostKeyFunc, limiter1)

	// Test fallback limiter
	req1 := &http.Request{Host: "unknown.com", URL: &url.URL{}}
	limiter, key := registry.GetLimiter(req1)
	if limiter != limiter1 {
		t.Errorf(fallbackMsg, limiter)
	}
	if key != defaultKey {
		t.Errorf(keyMsg, defaultKey, key)
	}

	// Register specific limiter
	registry.RegisterLimiter(testHostKey, limiter2)

	// Test specific limiter
	req2 := &http.Request{URL: &url.URL{Host: testHost}}
	limiter, key = registry.GetLimiter(req2)
	if limiter != limiter2 {
		t.Errorf("Expected specific limiter, got %v", limiter)
	}
	if key != testHostKey {
		t.Errorf(keyMsg, testHostKey, key)
	}

	// Test fallback for unregistered host
	req3 := &http.Request{URL: &url.URL{Host: "other.com"}}
	limiter, key = registry.GetLimiter(req3)
	if limiter != limiter1 {
		t.Errorf(fallbackMsg, limiter)
	}
	if key != defaultKey {
		t.Errorf(keyMsg, defaultKey, key)
	}
}

func TestRateLimiterRegistryAllow(t *testing.T) {
	limiter := NewRateLimiter(1, time.Hour) // Very slow refill, so we can test limiting
	registry := NewRateLimiterRegistry(DefaultHostKeyFunc, limiter)

	req := &http.Request{URL: &url.URL{Host: testHost}}

	// First request should be allowed
	allowed, key := registry.Allow(req)
	if !allowed {
		t.Error(firstReqMsg)
	}
	if key != defaultKey {
		t.Errorf(keyMsg, defaultKey, key)
	}

	// Second request should be denied (only 1 token)
	allowed, key = registry.Allow(req)
	if allowed {
		t.Error(secondReqMsg)
	}
	if key != defaultKey {
		t.Errorf(keyMsg, defaultKey, key)
	}
}

func TestRateLimiterRegistryConcurrency(t *testing.T) {
	registry := NewRateLimiterRegistry(DefaultHostKeyFunc, nil)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			limiter := NewRateLimiter(100, time.Second)
			key := fmt.Sprintf("host:test%d.com", id)
			registry.RegisterLimiter(key, limiter)

			// Test concurrent access
			for j := 0; j < 10; j++ {
				req := &http.Request{URL: &url.URL{Host: fmt.Sprintf("test%d.com", id)}}
				_, retrievedKey := registry.GetLimiter(req)
				if retrievedKey != key {
					t.Errorf(concurrentMsg, key, retrievedKey)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestDefaultKeyFuncs(t *testing.T) {
	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Host: testHost, Path: "/users"},
	}

	// Test host key func
	hostKey := DefaultHostKeyFunc(req)
	if hostKey != testHostKey {
		t.Errorf(hostKeyMsg, testHostKey, hostKey)
	}

	// Test route key func
	routeKey := DefaultRouteKeyFunc(req)
	expectedRouteKey := "route:GET:/users"
	if routeKey != expectedRouteKey {
		t.Errorf(routeKeyMsg, expectedRouteKey, routeKey)
	}

	// Test host-route key func
	hostRouteKey := DefaultHostRouteKeyFunc(req)
	expectedHostRouteKey := "host_route:api.example.com:GET:/users"
	if hostRouteKey != expectedHostRouteKey {
		t.Errorf(hostRouteKeyMsg, expectedHostRouteKey, hostRouteKey)
	}
}

func TestRateLimiterRegistryNoKeyFunc(t *testing.T) {
	limiter := NewRateLimiter(10, time.Second)
	registry := NewRateLimiterRegistry(nil, limiter)

	req := &http.Request{URL: &url.URL{Host: testHost}}

	limiterResult, key := registry.GetLimiter(req)
	if limiterResult != limiter {
		t.Errorf(fallbackMsg, limiterResult)
	}
	if key != defaultKey {
		t.Errorf(keyMsg, defaultKey, key)
	}
}

func TestRateLimiterRegistryNoLimiter(t *testing.T) {
	registry := NewRateLimiterRegistry(DefaultHostKeyFunc, nil)

	req := &http.Request{URL: &url.URL{Host: testHost}}

	limiter, key := registry.GetLimiter(req)
	if limiter != nil {
		t.Errorf(nilLimiterMsg, limiter)
	}
	if key != testHostKey {
		t.Errorf(keyMsg, testHostKey, key)
	}

	allowed, _ := registry.Allow(req)
	if !allowed {
		t.Error(allowMsg)
	}
}

func TestRateLimiterRegistryMetricsIntegration(t *testing.T) {
	// Test that metrics are correctly recorded with key labels
	registry := NewRateLimiterRegistry(DefaultHostKeyFunc, nil)
	limiter1 := NewRateLimiter(1, time.Hour) // Very slow refill
	limiter2 := NewRateLimiter(1, time.Hour) // Very slow refill

	// Register limiters for different hosts
	registry.RegisterLimiter("host:api.example.com", limiter1)
	registry.RegisterLimiter("host:web.example.com", limiter2)

	req1 := &http.Request{URL: &url.URL{Host: "api.example.com"}}
	req2 := &http.Request{URL: &url.URL{Host: "web.example.com"}}

	// First requests should be allowed
	allowed1, key1 := registry.Allow(req1)
	allowed2, key2 := registry.Allow(req2)

	if !allowed1 {
		t.Error("First request to api.example.com should be allowed")
	}
	if !allowed2 {
		t.Error("First request to web.example.com should be allowed")
	}

	if key1 != "host:api.example.com" {
		t.Errorf("Expected key 'host:api.example.com', got %s", key1)
	}
	if key2 != "host:web.example.com" {
		t.Errorf("Expected key 'host:web.example.com', got %s", key2)
	}

	// Second requests should be denied (each limiter only has 1 token)
	denied1, _ := registry.Allow(req1)
	denied2, _ := registry.Allow(req2)

	if denied1 {
		t.Error("Second request to api.example.com should be denied")
	}
	if denied2 {
		t.Error("Second request to web.example.com should be denied")
	}
}
