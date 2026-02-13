package ingestion

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/mr1hm/go-disaster-alerts/internal/config"
	"github.com/mr1hm/go-disaster-alerts/internal/models"
	"github.com/mr1hm/go-disaster-alerts/internal/repository"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

// mockDisasterRepo implements repository.DisasterRepository for testing
type mockDisasterRepo struct {
	mu        sync.Mutex
	disasters map[string]*models.Disaster
	addCount  atomic.Int64
}

func newMockRepo() *mockDisasterRepo {
	return &mockDisasterRepo{
		disasters: make(map[string]*models.Disaster),
	}
}

func (m *mockDisasterRepo) Add(ctx context.Context, d *models.Disaster) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.disasters[d.ID] = d
	m.addCount.Add(1)
	return nil
}

func (m *mockDisasterRepo) GetByID(ctx context.Context, id string) (*models.Disaster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.disasters[id], nil
}

func (m *mockDisasterRepo) Exists(ctx context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.disasters[id]
	return exists, nil
}

func (m *mockDisasterRepo) ListDisasters(ctx context.Context, opts repository.Filter) ([]models.Disaster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []models.Disaster
	for _, d := range m.disasters {
		results = append(results, *d)
	}
	return results, nil
}

func TestManager_StartStop(t *testing.T) {
	cfg := &config.Config{
		Worker: config.WorkerConfig{
			Count:      2,
			BufferSize: 10,
		},
		Sources: config.SourcesConfig{
			USGSEnabled:       false,
			GDACSEnabled:      false,
			USGSPollInterval:  time.Minute,
			GDACSPollInterval: time.Minute,
		},
	}

	repo := newMockRepo()
	mgr := NewManager(cfg, repo, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Start should not block
	mgr.Start(ctx)

	// Give it a moment
	time.Sleep(50 * time.Millisecond)

	// Cancel and stop
	cancel()
	mgr.Stop()

	// Should complete without hanging
}

func TestManager_ConcurrentSubmit(t *testing.T) {
	cfg := &config.Config{
		Worker: config.WorkerConfig{
			Count:      4,
			BufferSize: 100,
		},
		Sources: config.SourcesConfig{
			USGSEnabled:  false,
			GDACSEnabled: false,
		},
	}

	repo := newMockRepo()
	mgr := NewManager(cfg, repo, nil)

	ctx, cancel := context.WithCancel(context.Background())
	mgr.Start(ctx)

	// Submit many disasters concurrently
	var wg sync.WaitGroup
	numGoroutines := 10
	numPerGoroutine := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numPerGoroutine; j++ {
				d := &models.Disaster{
					ID:        fmt.Sprintf("test_%d_%d", goroutineID, j),
					Source:    "test",
					Type:      models.DisasterTypeEarthquake,
					Timestamp: time.Now(),
					CreatedAt: time.Now(),
				}
				mgr.pool.Submit(d)
			}
		}(i)
	}

	wg.Wait()

	// Give workers time to process
	time.Sleep(200 * time.Millisecond)

	cancel()
	mgr.Stop()

	// Verify all were processed
	expected := numGoroutines * numPerGoroutine
	actual := int(repo.addCount.Load())
	if actual != expected {
		t.Errorf("expected %d disasters added, got %d", expected, actual)
	}
}

func TestManager_GracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		Worker: config.WorkerConfig{
			Count:      2,
			BufferSize: 100,
		},
		Sources: config.SourcesConfig{
			USGSEnabled:  false,
			GDACSEnabled: false,
		},
	}

	repo := newMockRepo()
	mgr := NewManager(cfg, repo, nil)

	ctx, cancel := context.WithCancel(context.Background())
	mgr.Start(ctx)

	// Submit some work
	for i := 0; i < 50; i++ {
		d := &models.Disaster{
			ID:        fmt.Sprintf("shutdown_test_%d", i),
			Source:    "test",
			Type:      models.DisasterTypeFlood,
			Timestamp: time.Now(),
			CreatedAt: time.Now(),
		}
		mgr.pool.Submit(d)
	}

	// Immediately cancel
	cancel()

	// Stop should wait for in-flight work
	done := make(chan struct{})
	go func() {
		mgr.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good, stopped gracefully
	case <-time.After(5 * time.Second):
		t.Fatal("manager.Stop() timed out - possible goroutine leak")
	}
}

func TestManager_RaceCondition(t *testing.T) {
	// This test is designed to catch race conditions when run with -race flag
	cfg := &config.Config{
		Worker: config.WorkerConfig{
			Count:      8, // More workers = more contention
			BufferSize: 50,
		},
		Sources: config.SourcesConfig{
			USGSEnabled:  false,
			GDACSEnabled: false,
		},
	}

	repo := newMockRepo()
	mgr := NewManager(cfg, repo, nil)

	ctx, cancel := context.WithCancel(context.Background())
	mgr.Start(ctx)

	// Hammer it with concurrent operations
	var wg sync.WaitGroup

	// Writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				d := &models.Disaster{
					ID:        fmt.Sprintf("race_%d_%d", id, j),
					Source:    "test",
					Type:      models.DisasterTypeEarthquake,
					Magnitude: float64(j),
					Timestamp: time.Now(),
					CreatedAt: time.Now(),
				}
				mgr.pool.Submit(d)
			}
		}(i)
	}

	// Let it run
	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	cancel()
	mgr.Stop()

	// If we get here without race detector complaining, we're good
	t.Logf("processed %d disasters without race conditions", repo.addCount.Load())
}
