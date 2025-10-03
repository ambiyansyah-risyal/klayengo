package klayengo

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CacheDirectives represents parsed Cache-Control directives.
type CacheDirectives struct {
	NoStore              bool
	NoCache              bool
	MaxAge               *time.Duration
	SMaxAge              *time.Duration
	StaleWhileRevalidate *time.Duration
	MustRevalidate       bool
	Public               bool
	Private              bool
}

// parseCacheControl parses Cache-Control header into structured directives.
func parseCacheControl(header string) *CacheDirectives {
	directives := &CacheDirectives{}
	if header == "" {
		return directives
	}

	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Handle directives with values
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			value := strings.Trim(strings.TrimSpace(kv[1]), "\"")

			switch key {
			case "max-age":
				if seconds, err := strconv.Atoi(value); err == nil {
					maxAge := time.Duration(seconds) * time.Second
					directives.MaxAge = &maxAge
				}
			case "s-maxage":
				if seconds, err := strconv.Atoi(value); err == nil {
					sMaxAge := time.Duration(seconds) * time.Second
					directives.SMaxAge = &sMaxAge
				}
			case "stale-while-revalidate":
				if seconds, err := strconv.Atoi(value); err == nil {
					swr := time.Duration(seconds) * time.Second
					directives.StaleWhileRevalidate = &swr
				}
			}
		} else {
			// Handle boolean directives
			switch part {
			case "no-store":
				directives.NoStore = true
			case "no-cache":
				directives.NoCache = true
			case "must-revalidate":
				directives.MustRevalidate = true
			case "public":
				directives.Public = true
			case "private":
				directives.Private = true
			}
		}
	}

	return directives
}

// parseExpires parses the Expires header into a time.Time.
func parseExpires(header string) *time.Time {
	if header == "" {
		return nil
	}

	// Try RFC1123 format (preferred)
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		return &t
	}

	// Try RFC850 format
	if t, err := time.Parse(time.RFC850, header); err == nil {
		return &t
	}

	// Try ANSIC format
	if t, err := time.Parse(time.ANSIC, header); err == nil {
		return &t
	}

	return nil
}

// parseLastModified parses the Last-Modified header into a time.Time.
func parseLastModified(header string) *time.Time {
	if header == "" {
		return nil
	}

	// Try RFC1123 format (preferred)
	if t, err := time.Parse(time.RFC1123, header); err == nil {
		return &t
	}

	// Try RFC850 format
	if t, err := time.Parse(time.RFC850, header); err == nil {
		return &t
	}

	// Try ANSIC format
	if t, err := time.Parse(time.ANSIC, header); err == nil {
		return &t
	}

	return nil
}

// calculateCacheExpiry determines when a response expires based on HTTP headers.
func calculateCacheExpiry(resp *http.Response, receivedAt time.Time) (time.Time, bool) {
	cacheControl := parseCacheControl(resp.Header.Get("Cache-Control"))

	// If no-store or no-cache, don't cache
	if cacheControl.NoStore || cacheControl.NoCache {
		return time.Time{}, false
	}

	// Prefer max-age over Expires
	if cacheControl.MaxAge != nil {
		return receivedAt.Add(*cacheControl.MaxAge), true
	}

	// Fall back to Expires header
	if expiresTime := parseExpires(resp.Header.Get("Expires")); expiresTime != nil {
		return *expiresTime, true
	}

	// No explicit cache headers
	return time.Time{}, false
}

// shouldRevalidate determines if a cached response needs revalidation.
func shouldRevalidate(entry *CacheEntry, cacheControl *CacheDirectives) bool {
	if cacheControl == nil {
		return false
	}

	// Always revalidate if no-cache or must-revalidate
	if cacheControl.NoCache || cacheControl.MustRevalidate {
		return true
	}

	// Check if expired
	return time.Now().After(entry.ExpiresAt)
}

// addConditionalHeaders adds If-None-Match and If-Modified-Since headers to a request.
func addConditionalHeaders(req *http.Request, entry *CacheEntry) {
	if entry.ETag != "" {
		req.Header.Set("If-None-Match", entry.ETag)
	}
	if entry.LastModified != nil {
		req.Header.Set("If-Modified-Since", entry.LastModified.Format(time.RFC1123))
	}
}

// isNotModified checks if a response indicates the cached version is still valid.
func isNotModified(resp *http.Response) bool {
	return resp.StatusCode == http.StatusNotModified
}

// createEnhancedCacheEntry creates a CacheEntry with HTTP cache metadata.
func createEnhancedCacheEntry(resp *http.Response, body []byte, receivedAt time.Time) *CacheEntry {
	entry := &CacheEntry{
		Response:   resp,
		Body:       body,
		StatusCode: resp.StatusCode,
		Header:     resp.Header.Clone(), // Clone headers to avoid issues
	}

	// Parse ETag
	if etag := resp.Header.Get("ETag"); etag != "" {
		entry.ETag = etag
	}

	// Parse Last-Modified
	entry.LastModified = parseLastModified(resp.Header.Get("Last-Modified"))

	// Calculate expiry
	if expiryTime, shouldCache := calculateCacheExpiry(resp, receivedAt); shouldCache {
		entry.ExpiresAt = expiryTime

		// For SWR mode, calculate stale time
		cacheControl := parseCacheControl(resp.Header.Get("Cache-Control"))
		if cacheControl.StaleWhileRevalidate != nil {
			staleAt := expiryTime.Add(*cacheControl.StaleWhileRevalidate)
			entry.StaleAt = &staleAt
		}

		// Store max-age if present
		if cacheControl.MaxAge != nil {
			entry.MaxAge = cacheControl.MaxAge
		}
	}

	return entry
}