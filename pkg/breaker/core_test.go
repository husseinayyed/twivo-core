package breaker

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
)

func TestRegistryExecute(t *testing.T) {
	registry := NewGroup()
	registry.New("test", time.Second, time.Second)

	result, err := registry.Execute("test", func() (any, error) {
		return "value", nil
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "value" {
		t.Fatalf("Execute returned %v, want value", result)
	}

	wantErr := errors.New("operation failed")
	_, err = registry.Execute("test", func() (any, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute returned error %v, want %v", err, wantErr)
	}

	callbackCalled := false
	if _, err := registry.Execute("missing", func() (any, error) {
		callbackCalled = true
		return nil, nil
	}); err == nil {
		t.Fatal("Execute on an unregistered service returned no error")
	}
	if callbackCalled {
		t.Fatal("callback invoked for an unregistered service")
	}
}

func TestRegistryState(t *testing.T) {
	registry := NewGroup()
	if state := registry.State("missing"); state != gobreaker.StateClosed {
		t.Fatalf("State for an unregistered service = %v, want %v", state, gobreaker.StateClosed)
	}

	registry.New("test", time.Second, time.Second)
	if state := registry.State("test"); state != gobreaker.StateClosed {
		t.Fatalf("initial State = %v, want %v", state, gobreaker.StateClosed)
	}
}

func TestRegistryConcurrentAccess(t *testing.T) {
	registry := NewGroup()
	registry.New("test", time.Second, time.Second)

	const workers = 32
	const iterations = 100

	var callbackCount atomic.Int64
	start := make(chan struct{})
	errs := make(chan error, workers)
	var wait sync.WaitGroup

	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			<-start
			for iteration := 0; iteration < iterations; iteration++ {
				result, err := registry.Execute("test", func() (any, error) {
					return callbackCount.Add(1), nil
				})
				if err != nil {
					errs <- fmt.Errorf("worker %d Execute: %w", worker, err)
					return
				}
				if result == nil {
					errs <- fmt.Errorf("worker %d Execute returned a nil result", worker)
					return
				}
				if state := registry.State("test"); state != gobreaker.StateClosed {
					errs <- fmt.Errorf("worker %d observed state %v, want %v", worker, state, gobreaker.StateClosed)
					return
				}
			}
		}(worker)
	}

	// Re-registering while operations run exercises synchronized access to the registry map.
	wait.Add(1)
	go func() {
		defer wait.Done()
		<-start
		for i := 0; i < iterations; i++ {
			registry.New("test", time.Second, time.Second)
			registry.State("test")
		}
	}()

	close(start)
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	if got, want := callbackCount.Load(), int64(workers*iterations); got != want {
		t.Errorf("callback count = %d, want %d", got, want)
	}
}
