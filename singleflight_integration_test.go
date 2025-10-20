package klayengo

import (
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientSingleFlightIntegration(t *testing.T) {
	// Test that the new singleflight group can be enabled
	client := New(WithSingleFlightEnabled(true))

	if !client.singleFlightEnabled {
		t.Error("SingleFlight should be enabled")
	}

	if client.singleFlightGroup == nil {
		t.Error("SingleFlight group should be initialized")
	}
}

func TestClientSingleFlightDisabledByDefault(t *testing.T) {
	// Test that singleflight is disabled by default
	client := New()

	if client.singleFlightEnabled {
		t.Error("SingleFlight should be disabled by default")
	}

	if client.singleFlightGroup == nil {
		t.Error("SingleFlight group should still be initialized")
	}
}

func TestClientSingleFlightDoWithNewImplementation(t *testing.T) {
	// Test that when enabled, it uses the internal singleflight group
	client := New(WithSingleFlightEnabled(true))

	var callCount int64

	fn := func() (*http.Response, error) {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(10 * time.Millisecond) // Simulate work
		return &http.Response{StatusCode: 200}, nil
	}

	// Call multiple times concurrently with same key
	const numCalls = 10
	var wg sync.WaitGroup

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.singleFlightDo("test-key", fn)
		}()
	}

	wg.Wait()

	// Should only be called once due to singleflight
	if atomic.LoadInt64(&callCount) != 1 {
		t.Errorf("Function called %d times, want 1", atomic.LoadInt64(&callCount))
	}
}

func TestClientSingleFlightDoWithLegacyImplementation(t *testing.T) {
	// Test that when disabled, it uses the legacy implementation
	client := New(WithSingleFlightEnabled(false))

	var callCount int64

	fn := func() (*http.Response, error) {
		atomic.AddInt64(&callCount, 1)
		time.Sleep(10 * time.Millisecond) // Simulate work
		return &http.Response{StatusCode: 200}, nil
	}

	// Call multiple times concurrently with same key
	const numCalls = 10
	var wg sync.WaitGroup

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.singleFlightDo("test-key", fn)
		}()
	}

	wg.Wait()

	// Should only be called once due to legacy singleflight
	if atomic.LoadInt64(&callCount) != 1 {
		t.Errorf("Function called %d times, want 1", atomic.LoadInt64(&callCount))
	}
}

func BenchmarkSingleFlightDoNew(b *testing.B) {
	client := New(WithSingleFlightEnabled(true))

	fn := func() (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.singleFlightDo("bench-key", fn)
	}
}

func BenchmarkSingleFlightDoLegacy(b *testing.B) {
	client := New(WithSingleFlightEnabled(false))

	fn := func() (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.singleFlightDo("bench-key", fn)
	}
}
