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

// DeduplicationEntry represents an in-flight request shared between callers.
type DeduplicationEntry struct {
	response *http.Response
	err      error
	done     chan struct{}
	mu       sync.Mutex
	waiters  int
}

// DeduplicationTracker tracks in-flight requests to coalesce duplicates.
type DeduplicationTracker struct {
	mu      sync.RWMutex
	entries map[string]*DeduplicationEntry
}

// NewDeduplicationTracker returns an in-memory de-duplication tracker.
func NewDeduplicationTracker() *DeduplicationTracker {
	return &DeduplicationTracker{
		entries: make(map[string]*DeduplicationEntry),
	}
}

// GetOrCreateEntry returns an existing entry (not owner) or creates a new one (owner=true).
func (dt *DeduplicationTracker) GetOrCreateEntry(key string) (*DeduplicationEntry, bool) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	if entry, exists := dt.entries[key]; exists {
		entry.mu.Lock()
		entry.waiters++
		entry.mu.Unlock()
		return entry, false
	}

	entry := &DeduplicationEntry{
		done:    make(chan struct{}),
		waiters: 1,
	}
	dt.entries[key] = entry
	return entry, true
}

// Complete finalizes an entry and releases waiters.
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

	time.AfterFunc(100*time.Millisecond, func() {
		dt.mu.Lock()
		delete(dt.entries, key)
		dt.mu.Unlock()
	})
}

// Wait blocks until the owning request completes or context cancels.
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

// DeduplicationKeyFunc builds a key for identifying identical in-flight requests.
type DeduplicationKeyFunc func(*http.Request) string

// DefaultDeduplicationKeyFunc builds a key from method + URL (+ body hash for mutating verbs).
func DefaultDeduplicationKeyFunc(req *http.Request) string {
	h := fnv.New64a()
	h.Write([]byte(req.Method))
	h.Write([]byte(req.URL.String()))

	if req.Body != nil && (req.Method == "POST" || req.Method == "PUT" || req.Method == "PATCH") {
		bodyHash := sha256.New()
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err == nil {
				_, err := io.Copy(bodyHash, body)
				if err != nil {
					_ = err
				}
			}
		}
		h.Write(bodyHash.Sum(nil))
	}

	return fmt.Sprintf("%x", h.Sum64())
}

// DeduplicationCondition decides whether a request is eligible for deduplication.
type DeduplicationCondition func(req *http.Request) bool

// DefaultDeduplicationCondition enables deduplication for safe idempotent methods.
func DefaultDeduplicationCondition(req *http.Request) bool {
	return req.Method == "GET" || req.Method == "HEAD" || req.Method == "OPTIONS"
}
