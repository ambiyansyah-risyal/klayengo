package klayengo

import (
	"net/http"
	"sync"
)

// NewRateLimiterRegistry creates a new rate limiter registry with the given key function and fallback limiter.
func NewRateLimiterRegistry(keyFunc KeyFunc, fallback Limiter) *RateLimiterRegistry {
	return &RateLimiterRegistry{
		limiters: make(map[string]Limiter),
		keyFunc:  keyFunc,
		fallback: fallback,
		mutex:    sync.RWMutex{},
	}
}

// RegisterLimiter adds a limiter for the given key.
func (r *RateLimiterRegistry) RegisterLimiter(key string, limiter Limiter) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.limiters[key] = limiter
}

// GetLimiter returns the limiter for the given request, using the key function to determine the key.
// If no specific limiter is found, returns the fallback limiter.
func (r *RateLimiterRegistry) GetLimiter(req *http.Request) (Limiter, string) {
	if r.keyFunc == nil {
		if r.fallback != nil {
			return r.fallback, "default"
		}
		return nil, "default"
	}

	key := r.keyFunc(req)

	r.mutex.RLock()
	limiter, exists := r.limiters[key]
	r.mutex.RUnlock()

	if exists {
		return limiter, key
	}

	if r.fallback != nil {
		return r.fallback, "default"
	}

	return nil, key
}

// Allow checks if a request is allowed by the appropriate rate limiter.
func (r *RateLimiterRegistry) Allow(req *http.Request) (bool, string) {
	limiter, key := r.GetLimiter(req)
	if limiter == nil {
		return true, key // No rate limiting if no limiter found
	}
	return limiter.Allow(), key
}

// DefaultHostKeyFunc generates a key based on the request host.
func DefaultHostKeyFunc(req *http.Request) string {
	if req.URL.Host != "" {
		return "host:" + req.URL.Host
	}
	if req.Host != "" {
		return "host:" + req.Host
	}
	return "host:unknown"
}

// DefaultRouteKeyFunc generates a key based on the request method and path.
func DefaultRouteKeyFunc(req *http.Request) string {
	return "route:" + req.Method + ":" + req.URL.Path
}

// DefaultHostRouteKeyFunc generates a key combining host and route.
func DefaultHostRouteKeyFunc(req *http.Request) string {
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}
	if host == "" {
		host = "unknown"
	}
	return "host_route:" + host + ":" + req.Method + ":" + req.URL.Path
}
