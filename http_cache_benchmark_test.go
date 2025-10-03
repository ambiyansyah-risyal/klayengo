package klayengo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkCacheModes compares performance between different cache modes
func BenchmarkCacheModes(b *testing.B) {
	callCount := int64(0)
	
	// Server with cache-friendly headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&callCount, 1)
		w.Header().Set("ETag", "\"benchmark-etag\"")
		w.Header().Set("Cache-Control", "max-age=3600")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("benchmark response"))
	}))
	defer server.Close()

	b.Run("TTLOnly", func(b *testing.B) {
		atomic.StoreInt64(&callCount, 0)
		client := New(WithCache(1 * time.Hour))
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(ctx, server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
		
		finalCount := atomic.LoadInt64(&callCount)
		b.Logf("Server calls: %d/%d (%.2f%% cache hit rate)", finalCount, b.N, float64(b.N-int(finalCount))/float64(b.N)*100)
	})

	b.Run("HTTPSemantics", func(b *testing.B) {
		atomic.StoreInt64(&callCount, 0)
		cache := NewInMemoryCache()
		provider := NewHTTPSemanticsCacheProvider(cache, 1*time.Hour, HTTPSemantics)
		client := New(
			WithCacheProvider(provider),
			WithCacheMode(HTTPSemantics),
		)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(ctx, server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
		
		finalCount := atomic.LoadInt64(&callCount)
		b.Logf("Server calls: %d/%d (%.2f%% cache hit rate)", finalCount, b.N, float64(b.N-int(finalCount))/float64(b.N)*100)
	})

	b.Run("SWR", func(b *testing.B) {
		atomic.StoreInt64(&callCount, 0)
		cache := NewInMemoryCache()
		provider := NewHTTPSemanticsCacheProvider(cache, 1*time.Hour, SWR)
		client := New(
			WithCacheProvider(provider),
			WithCacheMode(SWR),
		)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			resp, err := client.Get(ctx, server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
		
		finalCount := atomic.LoadInt64(&callCount)
		b.Logf("Server calls: %d/%d (%.2f%% cache hit rate)", finalCount, b.N, float64(b.N-int(finalCount))/float64(b.N)*100)
	})
}

// BenchmarkSingleFlight demonstrates stampede protection
func BenchmarkSingleFlight(b *testing.B) {
	callCount := int64(0)
	
	// Slow server to ensure concurrent requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(10 * time.Millisecond) // Simulate slow response
		w.Header().Set("Cache-Control", "max-age=3600")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response"))
	}))
	defer server.Close()

	b.Run("WithoutSingleFlight", func(b *testing.B) {
		atomic.StoreInt64(&callCount, 0)
		client := New() // No cache, no single-flight
		ctx := context.Background()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(ctx, server.URL)
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()
			}
		})
		
		finalCount := atomic.LoadInt64(&callCount)
		b.Logf("Server calls: %d/%d", finalCount, b.N)
	})

	b.Run("WithSingleFlight", func(b *testing.B) {
		atomic.StoreInt64(&callCount, 0)
		cache := NewInMemoryCache()
		provider := NewHTTPSemanticsCacheProvider(cache, 1*time.Hour, HTTPSemantics)
		client := New(
			WithCacheProvider(provider),
			WithCacheMode(HTTPSemantics),
		)
		ctx := context.Background()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := client.Get(ctx, server.URL)
				if err != nil {
					b.Fatal(err)
				}
				resp.Body.Close()
			}
		})
		
		finalCount := atomic.LoadInt64(&callCount)
		b.Logf("Server calls: %d/%d (%.2f%% reduction)", finalCount, b.N, (1-float64(finalCount)/float64(b.N))*100)
	})
}

// BenchmarkConditionalRequests measures overhead of conditional requests
func BenchmarkConditionalRequests(b *testing.B) {
	callCount := int64(0)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt64(&callCount, 1)
		
		etag := "\"benchmark-etag\""
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "max-age=1") // Short TTL to force revalidation
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("response %d", count)))
	}))
	defer server.Close()

	cache := NewInMemoryCache()
	provider := NewHTTPSemanticsCacheProvider(cache, 1*time.Hour, HTTPSemantics)
	client := New(
		WithCacheProvider(provider),
		WithCacheMode(HTTPSemantics),
	)
	
	ctx := context.Background()
	
	// Prime the cache
	resp, err := client.Get(ctx, server.URL)
	if err != nil {
		b.Fatal(err)
	}
	resp.Body.Close()
	
	// Wait for entry to become stale to trigger conditional requests
	time.Sleep(2 * time.Second)
	atomic.StoreInt64(&callCount, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(ctx, server.URL)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
	
	finalCount := atomic.LoadInt64(&callCount)
	b.Logf("Server calls: %d/%d (conditional requests)", finalCount, b.N)
}

// BenchmarkHeaderParsing measures cache header parsing overhead
func BenchmarkHeaderParsing(b *testing.B) {
	headers := []string{
		"max-age=3600",
		"max-age=3600, public",
		"max-age=3600, stale-while-revalidate=300",
		"max-age=3600, no-cache, must-revalidate",
		"no-store",
		"max-age=0, must-revalidate, private",
	}

	b.Run("ParseCacheControl", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			header := headers[i%len(headers)]
			_ = parseCacheControl(header)
		}
	})

	expires := []string{
		"Wed, 21 Oct 2015 07:28:00 GMT",
		"Thu, 01 Dec 1994 16:00:00 GMT",
		"Fri, 30 Oct 1998 14:19:41 GMT",
	}

	b.Run("ParseExpires", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			header := expires[i%len(expires)]
			_ = parseExpires(header)
		}
	})
}