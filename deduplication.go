package klayengo

import (
	"context"
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"sync"
	"time"
)

// DeduplicationEntry represents an in-flight request for deduplication
type DeduplicationEntry struct {
	response *http.Response
	err      error
	done     chan struct{}
	mu       sync.Mutex
	waiters  int
}

// DeduplicationTracker tracks in-flight requests for deduplication
type DeduplicationTracker struct {
	mu      sync.RWMutex
	entries map[string]*DeduplicationEntry
}

// NewDeduplicationTracker creates a new deduplication tracker
func NewDeduplicationTracker() *DeduplicationTracker {
	return &DeduplicationTracker{
		entries: make(map[string]*DeduplicationEntry),
	}
}

// GetOrCreateEntry gets an existing entry or creates a new one for the given key
// Returns the entry and a boolean indicating if this caller is the owner (should make the request)
func (dt *DeduplicationTracker) GetOrCreateEntry(key string) (*DeduplicationEntry, bool) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if entry, exists := dt.entries[key]; exists {
		entry.mu.Lock()
		entry.waiters++
		entry.mu.Unlock()
		return entry, false // not the owner
	}

	entry := &DeduplicationEntry{
		done:    make(chan struct{}),
		waiters: 1,
	}
	dt.entries[key] = entry
	return entry, true // is the owner
}

// Complete marks the entry as completed with the given response and error
func (dt *DeduplicationTracker) Complete(key string, resp *http.Response, err error) {
	dt.mu.Lock()
	entry, exists := dt.entries[key]
	dt.mu.Unlock()

	if !exists {
		return
	}

	entry.mu.Lock()
	entry.response = resp
	entry.err = err
	close(entry.done)
	entry.mu.Unlock()

	// Clean up after a short delay to allow waiters to get the result
	time.AfterFunc(100*time.Millisecond, func() {
		dt.mu.Lock()
		delete(dt.entries, key)
		dt.mu.Unlock()
	})
}

// Wait waits for the entry to complete and returns the response and error
func (entry *DeduplicationEntry) Wait(ctx context.Context) (*http.Response, error) {
	select {
	case <-entry.done:
		entry.mu.Lock()
		resp := entry.response
		err := entry.err
		entry.mu.Unlock()
		return resp, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// DeduplicationKeyFunc generates a deduplication key for a request
type DeduplicationKeyFunc func(*http.Request) string

// DefaultDeduplicationKeyFunc generates a default deduplication key
func DefaultDeduplicationKeyFunc(req *http.Request) string {
	h := fnv.New64a()
	h.Write([]byte(req.Method))
	h.Write([]byte(req.URL.String()))

	// For requests with bodies, include a hash of the body
	if req.Body != nil && (req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH") {
		bodyHash := sha256.New()
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err == nil {
				_, err := io.Copy(bodyHash, body)
				if err != nil {
					// If body reading fails, continue without body hash
					// This ensures deduplication still works for requests without bodies
					_ = err // Explicitly ignore the error as per linter requirement
				}
			}
		}
		h.Write(bodyHash.Sum(nil))
	}

	return fmt.Sprintf("%x", h.Sum64())
}

// DeduplicationCondition determines whether a request should be deduplicated
type DeduplicationCondition func(req *http.Request) bool

// DefaultDeduplicationCondition is the default condition for deduplication
func DefaultDeduplicationCondition(req *http.Request) bool {
	// Only deduplicate GET, HEAD, and OPTIONS requests by default
	// POST/PUT/PATCH should not be deduplicated unless explicitly configured
	return req.Method == "GET" || req.Method == "HEAD" || req.Method == "OPTIONS"
}
