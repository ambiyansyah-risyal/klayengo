package klayengo

import (
	"net/http"
	"testing"
	"time"
)

func TestParseCacheControl(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected *CacheDirectives
	}{
		{
			name:     "empty header",
			header:   "",
			expected: &CacheDirectives{},
		},
		{
			name:   "max-age directive",
			header: "max-age=3600",
			expected: &CacheDirectives{
				MaxAge: func() *time.Duration { d := 3600 * time.Second; return &d }(),
			},
		},
		{
			name:   "multiple directives",
			header: "max-age=3600, no-cache, public",
			expected: &CacheDirectives{
				MaxAge:  func() *time.Duration { d := 3600 * time.Second; return &d }(),
				NoCache: true,
				Public:  true,
			},
		},
		{
			name:   "stale-while-revalidate",
			header: "max-age=600, stale-while-revalidate=300",
			expected: &CacheDirectives{
				MaxAge:               func() *time.Duration { d := 600 * time.Second; return &d }(),
				StaleWhileRevalidate: func() *time.Duration { d := 300 * time.Second; return &d }(),
			},
		},
		{
			name:   "no-store directive",
			header: "no-store",
			expected: &CacheDirectives{
				NoStore: true,
			},
		},
		{
			name:   "must-revalidate directive",
			header: "max-age=0, must-revalidate",
			expected: &CacheDirectives{
				MaxAge:         func() *time.Duration { d := 0 * time.Second; return &d }(),
				MustRevalidate: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCacheControl(tt.header)

			if result.NoStore != tt.expected.NoStore {
				t.Errorf("NoStore: expected %v, got %v", tt.expected.NoStore, result.NoStore)
			}

			if result.NoCache != tt.expected.NoCache {
				t.Errorf("NoCache: expected %v, got %v", tt.expected.NoCache, result.NoCache)
			}

			if result.MustRevalidate != tt.expected.MustRevalidate {
				t.Errorf("MustRevalidate: expected %v, got %v", tt.expected.MustRevalidate, result.MustRevalidate)
			}

			if result.Public != tt.expected.Public {
				t.Errorf("Public: expected %v, got %v", tt.expected.Public, result.Public)
			}

			if result.Private != tt.expected.Private {
				t.Errorf("Private: expected %v, got %v", tt.expected.Private, result.Private)
			}

			// Check MaxAge
			if tt.expected.MaxAge != nil {
				if result.MaxAge == nil || *result.MaxAge != *tt.expected.MaxAge {
					t.Errorf("MaxAge: expected %v, got %v", tt.expected.MaxAge, result.MaxAge)
				}
			} else if result.MaxAge != nil {
				t.Errorf("MaxAge: expected nil, got %v", result.MaxAge)
			}

			// Check StaleWhileRevalidate
			if tt.expected.StaleWhileRevalidate != nil {
				if result.StaleWhileRevalidate == nil || *result.StaleWhileRevalidate != *tt.expected.StaleWhileRevalidate {
					t.Errorf("StaleWhileRevalidate: expected %v, got %v", tt.expected.StaleWhileRevalidate, result.StaleWhileRevalidate)
				}
			} else if result.StaleWhileRevalidate != nil {
				t.Errorf("StaleWhileRevalidate: expected nil, got %v", result.StaleWhileRevalidate)
			}
		})
	}
}

func TestParseExpires(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected bool
	}{
		{
			name:     "empty header",
			header:   "",
			expected: false,
		},
		{
			name:     "valid RFC1123 format",
			header:   "Wed, 21 Oct 2015 07:28:00 GMT",
			expected: true,
		},
		{
			name:     "invalid format",
			header:   "invalid-date",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseExpires(tt.header)
			if (result != nil) != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result != nil)
			}
		})
	}
}

func TestParseLastModified(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected bool
	}{
		{
			name:     "empty header",
			header:   "",
			expected: false,
		},
		{
			name:     "valid RFC1123 format",
			header:   "Wed, 21 Oct 2015 07:28:00 GMT",
			expected: true,
		},
		{
			name:     "invalid format",
			header:   "invalid-date",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLastModified(tt.header)
			if (result != nil) != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result != nil)
			}
		})
	}
}

func TestCalculateCacheExpiry(t *testing.T) {
	receivedAt := time.Now()

	tests := []struct {
		name         string
		headers      map[string]string
		shouldCache  bool
		expectedDiff time.Duration
	}{
		{
			name:        "no cache headers",
			headers:     map[string]string{},
			shouldCache: false,
		},
		{
			name: "no-store directive",
			headers: map[string]string{
				"Cache-Control": "no-store",
			},
			shouldCache: false,
		},
		{
			name: "no-cache directive",
			headers: map[string]string{
				"Cache-Control": "no-cache",
			},
			shouldCache: false,
		},
		{
			name: "max-age directive",
			headers: map[string]string{
				"Cache-Control": "max-age=3600",
			},
			shouldCache:  true,
			expectedDiff: 3600 * time.Second,
		},
		{
			name: "expires header",
			headers: map[string]string{
				"Expires": receivedAt.Add(2 * time.Hour).Format(time.RFC1123),
			},
			shouldCache:  true,
			expectedDiff: 2 * time.Hour,
		},
		{
			name: "max-age takes precedence over expires",
			headers: map[string]string{
				"Cache-Control": "max-age=1800",
				"Expires":       receivedAt.Add(2 * time.Hour).Format(time.RFC1123),
			},
			shouldCache:  true,
			expectedDiff: 1800 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: make(http.Header),
			}

			for key, value := range tt.headers {
				resp.Header.Set(key, value)
			}

			expiryTime, shouldCache := calculateCacheExpiry(resp, receivedAt)

			if shouldCache != tt.shouldCache {
				t.Errorf("Expected shouldCache %v, got %v", tt.shouldCache, shouldCache)
				return
			}

			if tt.shouldCache {
				actualDiff := expiryTime.Sub(receivedAt)
				// Allow small tolerance for time differences
				tolerance := 1 * time.Second
				if actualDiff < tt.expectedDiff-tolerance || actualDiff > tt.expectedDiff+tolerance {
					t.Errorf("Expected expiry diff ~%v, got %v", tt.expectedDiff, actualDiff)
				}
			}
		})
	}
}

func TestShouldRevalidate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name              string
		entry             *CacheEntry
		cacheControlValue string
		expected          bool
	}{
		{
			name: "no cache control",
			entry: &CacheEntry{
				ExpiresAt: now.Add(1 * time.Hour),
			},
			cacheControlValue: "",
			expected:          false,
		},
		{
			name: "no-cache directive",
			entry: &CacheEntry{
				ExpiresAt: now.Add(1 * time.Hour),
				Header:    http.Header{"Cache-Control": []string{"no-cache"}},
			},
			cacheControlValue: "no-cache",
			expected:          true,
		},
		{
			name: "must-revalidate directive",
			entry: &CacheEntry{
				ExpiresAt: now.Add(1 * time.Hour),
				Header:    http.Header{"Cache-Control": []string{"must-revalidate"}},
			},
			cacheControlValue: "must-revalidate",
			expected:          true,
		},
		{
			name: "expired entry",
			entry: &CacheEntry{
				ExpiresAt: now.Add(-1 * time.Hour), // expired
			},
			cacheControlValue: "max-age=3600",
			expected:          true,
		},
		{
			name: "fresh entry",
			entry: &CacheEntry{
				ExpiresAt: now.Add(1 * time.Hour), // fresh
			},
			cacheControlValue: "max-age=3600",
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cacheControl *CacheDirectives
			if tt.cacheControlValue != "" {
				cacheControl = parseCacheControl(tt.cacheControlValue)
			}

			result := shouldRevalidate(tt.entry, cacheControl)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAddConditionalHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	lastModified := time.Now().Add(-1 * time.Hour)

	entry := &CacheEntry{
		ETag:         "\"123456\"",
		LastModified: &lastModified,
	}

	addConditionalHeaders(req, entry)

	if req.Header.Get("If-None-Match") != "\"123456\"" {
		t.Errorf("Expected If-None-Match header to be set to \"123456\", got %s", req.Header.Get("If-None-Match"))
	}

	expectedLastModified := lastModified.Format(time.RFC1123)
	if req.Header.Get("If-Modified-Since") != expectedLastModified {
		t.Errorf("Expected If-Modified-Since header to be %s, got %s", expectedLastModified, req.Header.Get("If-Modified-Since"))
	}
}

func TestIsNotModified(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   bool
	}{
		{
			name:       "304 Not Modified",
			statusCode: http.StatusNotModified,
			expected:   true,
		},
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
			expected:   false,
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
			}

			result := isNotModified(resp)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCreateEnhancedCacheEntry(t *testing.T) {
	receivedAt := time.Now()
	lastModified := receivedAt.Add(-1 * time.Hour)

	resp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
	}
	resp.Header.Set("ETag", "\"123456\"")
	resp.Header.Set("Last-Modified", lastModified.Format(time.RFC1123))
	resp.Header.Set("Cache-Control", "max-age=3600, stale-while-revalidate=300")

	body := []byte("test response body")
	entry := createEnhancedCacheEntry(resp, body, receivedAt)

	if entry.ETag != "\"123456\"" {
		t.Errorf("Expected ETag to be \"\\\"123456\\\"\", got \"%s\"", entry.ETag)
	}

	if entry.LastModified == nil {
		t.Error("Expected LastModified to be set")
	} else {
		// RFC1123 format loses sub-second precision, so compare truncated times
		expectedTime := lastModified.Truncate(time.Second)
		actualTime := entry.LastModified.Truncate(time.Second)
		if !expectedTime.Equal(actualTime) {
			t.Errorf("Expected LastModified to be %v, got %v", expectedTime, actualTime)
		}
	}

	expectedExpiry := receivedAt.Add(3600 * time.Second)
	if !entry.ExpiresAt.Equal(expectedExpiry) {
		t.Errorf("Expected ExpiresAt to be %v, got %v", expectedExpiry, entry.ExpiresAt)
	}

	if entry.MaxAge == nil || *entry.MaxAge != 3600*time.Second {
		t.Errorf("Expected MaxAge to be 3600s, got %v", entry.MaxAge)
	}

	if entry.StaleAt == nil {
		t.Error("Expected StaleAt to be set")
	} else {
		expectedStaleAt := expectedExpiry.Add(300 * time.Second)
		if !entry.StaleAt.Equal(expectedStaleAt) {
			t.Errorf("Expected StaleAt to be %v, got %v", expectedStaleAt, *entry.StaleAt)
		}
	}
}
