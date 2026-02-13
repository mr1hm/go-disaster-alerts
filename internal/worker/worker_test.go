package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestWorkerPool_StartStop(t *testing.T) {
	var processed atomic.Int64
	processor := func(ctx context.Context, job Job) error {
		processed.Add(1)
		return nil
	}

	pool := NewWorkerPool(2, 10, processor)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Submit some jobs
	for i := 0; i < 5; i++ {
		pool.Submit(i)
	}

	time.Sleep(50 * time.Millisecond)

	cancel()
	pool.Stop()

	if processed.Load() != 5 {
		t.Errorf("expected 5 jobs processed, got %d", processed.Load())
	}
}

func TestWorkerPool_ConcurrentSubmit(t *testing.T) {
	var processed atomic.Int64
	processor := func(ctx context.Context, job Job) error {
		processed.Add(1)
		return nil
	}

	pool := NewWorkerPool(4, 100, processor)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Submit many jobs concurrently
	for i := 0; i < 100; i++ {
		go func(n int) {
			pool.Submit(n)
		}(i)
	}

	time.Sleep(100 * time.Millisecond)

	cancel()
	pool.Stop()

	if processed.Load() != 100 {
		t.Errorf("expected 100 jobs processed, got %d", processed.Load())
	}
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	var processed atomic.Int64
	processor := func(ctx context.Context, job Job) error {
		time.Sleep(10 * time.Millisecond) // Simulate work
		processed.Add(1)
		return nil
	}

	pool := NewWorkerPool(2, 50, processor)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Submit jobs
	for i := 0; i < 20; i++ {
		pool.Submit(i)
	}

	// Cancel immediately
	cancel()

	// Stop should wait for in-flight jobs
	done := make(chan struct{})
	go func() {
		pool.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good
	case <-time.After(5 * time.Second):
		t.Fatal("pool.Stop() timed out")
	}

	t.Logf("processed %d jobs before shutdown", processed.Load())
}

func TestWorkerPool_ContextCancellation(t *testing.T) {
	var started atomic.Int64
	var completed atomic.Int64

	processor := func(ctx context.Context, job Job) error {
		started.Add(1)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			completed.Add(1)
			return nil
		}
	}

	pool := NewWorkerPool(2, 10, processor)

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	// Submit jobs
	for i := 0; i < 5; i++ {
		pool.Submit(i)
	}

	// Wait a bit then cancel
	time.Sleep(50 * time.Millisecond)
	cancel()
	pool.Stop()

	t.Logf("started: %d, completed: %d", started.Load(), completed.Load())
}
