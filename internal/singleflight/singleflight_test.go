package singleflight

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	g := New()
	if g == nil {
		t.Fatal("New() returned nil")
	}
	if g.m == nil {
		t.Error("New() did not initialize map")
	}
}

func TestDo(t *testing.T) {
	g := New()

	// Test basic functionality
	val, err := g.Do("key1", func() (interface{}, error) {
		return "hello", nil
	})

	if err != nil {
		t.Errorf("Do() returned error: %v", err)
	}
	if val != "hello" {
		t.Errorf("Do() returned %v, want hello", val)
	}
}

func TestDoError(t *testing.T) {
	g := New()
	expectedErr := errors.New("test error")

	val, err := g.Do("key1", func() (interface{}, error) {
		return nil, expectedErr
	})

	if err != expectedErr {
		t.Errorf("Do() returned error %v, want %v", err, expectedErr)
	}
	if val != nil {
		t.Errorf("Do() returned %v, want nil", val)
	}
}

func TestDoDuplicateCalls(t *testing.T) {
	g := New()

	var callCount int
	var mu sync.Mutex

	// Function that increments call count
	fn := func() (interface{}, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(10 * time.Millisecond) // Simulate work
		return "result", nil
	}

	// Start multiple goroutines calling the same key
	const numCalls = 10
	var wg sync.WaitGroup
	results := make([]interface{}, numCalls)
	errors := make([]error, numCalls)

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index], errors[index] = g.Do("same-key", fn)
		}(i)
	}

	wg.Wait()

	// Verify only called once
	mu.Lock()
	if callCount != 1 {
		t.Errorf("Function called %d times, want 1", callCount)
	}
	mu.Unlock()

	// Verify all got the same result
	for i, result := range results {
		if errors[i] != nil {
			t.Errorf("Call %d returned error: %v", i, errors[i])
		}
		if result != "result" {
			t.Errorf("Call %d returned %v, want result", i, result)
		}
	}
}

func TestTryDo(t *testing.T) {
	g := New()

	// First call should succeed
	val, err, ok := g.TryDo("key1", func() (interface{}, error) {
		return "hello", nil
	})

	if !ok {
		t.Error("TryDo() should have succeeded")
	}
	if err != nil {
		t.Errorf("TryDo() returned error: %v", err)
	}
	if val != "hello" {
		t.Errorf("TryDo() returned %v, want hello", val)
	}
}

func TestTryDoInProgress(t *testing.T) {
	g := New()

	var started sync.WaitGroup
	var proceed sync.WaitGroup
	started.Add(1)
	proceed.Add(1)

	// Start a long-running operation
	go func() {
		_, _, _ = g.TryDo("key1", func() (interface{}, error) {
			started.Done()
			proceed.Wait() // Wait for test to proceed
			return "first", nil
		})
	}()

	started.Wait() // Wait for first call to start

	// Second call should return ErrInProgress
	val, err, ok := g.TryDo("key1", func() (interface{}, error) {
		return "second", nil
	})

	if ok {
		t.Error("TryDo() should have failed due to in-progress call")
	}
	if err != ErrInProgress {
		t.Errorf("TryDo() returned error %v, want %v", err, ErrInProgress)
	}
	if val != nil {
		t.Errorf("TryDo() returned %v, want nil", val)
	}

	proceed.Done() // Allow first call to complete
}

func TestForgetKey(t *testing.T) {
	g := New()

	// Add a key
	_, _ = g.Do("key1", func() (interface{}, error) {
		return "value", nil
	})

	// Forget the key
	g.ForgetKey("key1")

	// The key should be gone from the map (though this is implementation detail)
	// We can't directly test the map since it's private, but we can test behavior
	// A new call should work normally
	val, err := g.Do("key1", func() (interface{}, error) {
		return "new-value", nil
	})

	if err != nil {
		t.Errorf("Do() after ForgetKey returned error: %v", err)
	}
	if val != "new-value" {
		t.Errorf("Do() after ForgetKey returned %v, want new-value", val)
	}
}

func TestErrInProgress(t *testing.T) {
	if ErrInProgress == nil {
		t.Error("ErrInProgress should not be nil")
	}

	expected := "singleflight: call already in progress"
	if ErrInProgress.Error() != expected {
		t.Errorf("ErrInProgress.Error() = %q, want %q", ErrInProgress.Error(), expected)
	}
}

func BenchmarkDo(b *testing.B) {
	g := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.Do("bench-key", func() (interface{}, error) {
			return "result", nil
		})
	}
}

func BenchmarkTryDo(b *testing.B) {
	g := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = g.TryDo("bench-key", func() (interface{}, error) {
			return "result", nil
		})
	}
}
