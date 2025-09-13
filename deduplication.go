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

type DeduplicationEntry struct {
	response *http.Response
	err      error
	done     chan struct{}
	mu       sync.Mutex
	waiters  int
}

type DeduplicationTracker struct {
	mu      sync.RWMutex
	entries map[string]*DeduplicationEntry
}

func NewDeduplicationTracker() *DeduplicationTracker {
	return &DeduplicationTracker{
		entries: make(map[string]*DeduplicationEntry),
	}
}

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

type DeduplicationKeyFunc func(*http.Request) string

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

type DeduplicationCondition func(req *http.Request) bool

func DefaultDeduplicationCondition(req *http.Request) bool {
	return req.Method == "GET" || req.Method == "HEAD" || req.Method == "OPTIONS"
}
