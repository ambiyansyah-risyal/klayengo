package singleflight

import (
	"sync"
	"time"
)

// Group manages a set of in-flight calls to prevent duplicate work.
// This is a minimal implementation focused on owner/waiter semantics for cache SWR.
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

// call represents an active or completed function call.
type call struct {
	wg   sync.WaitGroup
	val  interface{}
	err  error
	done bool
}

// New creates a new singleflight Group.
func New() *Group {
	return &Group{
		m: make(map[string]*call),
	}
}

// Do executes and returns the results of the given function, making sure that
// only one execution is in-flight for a given key at a time. If a duplicate
// comes in, the duplicate caller waits for the original to complete and
// receives the same results.
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}

	c := &call{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.done = true
	c.wg.Done()

	// Clean up the call from the map after a short delay to allow
	// for immediate duplicate detection while preventing memory leaks.
	go func() {
		time.Sleep(100 * time.Millisecond)
		g.mu.Lock()
		if g.m[key] == c {
			delete(g.m, key)
		}
		g.mu.Unlock()
	}()

	return c.val, c.err
}

// TryDo attempts to execute the function only if no other call with the same key
// is in progress. If another call is in progress, it returns immediately with
// ErrInProgress. This is useful for non-blocking scenarios.
func (g *Group) TryDo(key string, fn func() (interface{}, error)) (interface{}, error, bool) {
	g.mu.Lock()
	if _, ok := g.m[key]; ok {
		g.mu.Unlock()
		return nil, ErrInProgress, false
	}

	c := &call{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.done = true
	c.wg.Done()

	// Clean up
	go func() {
		time.Sleep(100 * time.Millisecond)
		g.mu.Lock()
		if g.m[key] == c {
			delete(g.m, key)
		}
		g.mu.Unlock()
	}()

	return c.val, c.err, true
}

// ForgetKey removes the key from the group's map, effectively allowing
// future calls with the same key to execute even if a previous call
// is still in progress (though this should be used carefully).
func (g *Group) ForgetKey(key string) {
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()
}