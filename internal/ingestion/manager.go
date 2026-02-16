package ingestion

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/mr1hm/go-disaster-alerts/internal/config"
	internalgrpc "github.com/mr1hm/go-disaster-alerts/internal/grpc"
	"github.com/mr1hm/go-disaster-alerts/internal/models"
	"github.com/mr1hm/go-disaster-alerts/internal/repository"
	"github.com/mr1hm/go-disaster-alerts/internal/worker"
)

type Manager struct {
	cfg         *config.Config
	repo        repository.DisasterRepository
	broadcaster *internalgrpc.Broadcaster
	pool        *worker.WorkerPool
	wg          sync.WaitGroup
}

func NewManager(cfg *config.Config, repo repository.DisasterRepository, broadcaster *internalgrpc.Broadcaster) *Manager {
	return &Manager{
		cfg:         cfg,
		repo:        repo,
		broadcaster: broadcaster,
	}
}

func (m *Manager) Start(ctx context.Context) {
	processor := func(ctx context.Context, job worker.Job) error {
		disaster := job.(*models.Disaster)

		if err := m.repo.Add(ctx, disaster); err != nil {
			slog.Error("error adding disaster", "id", disaster.ID, "error", err)
			return err
		}

		// Broadcast to gRPC stream subscribers
		if m.broadcaster != nil {
			m.broadcaster.Broadcast(disaster)
		}

		slog.Info("added disaster", "id", disaster.ID, "type", disaster.Type, "source", disaster.Source, "alert_level", disaster.AlertLevel, "country", disaster.Country, "population", disaster.Population)
		return nil
	}

	m.pool = worker.NewWorkerPool(m.cfg.Worker.Count, m.cfg.Worker.BufferSize, processor)
	m.pool.Start(ctx)

	// Start GDACS poller if enabled
	if m.cfg.Sources.GDACSEnabled {
		m.wg.Add(1)
		go m.runPoller(ctx, "gdacs", m.cfg.Sources.GDACSURL, m.cfg.Sources.GDACSPollInterval)
	}
}

func (m *Manager) runPoller(ctx context.Context, source, url string, interval time.Duration) {
	defer m.wg.Done()
	slog.Info("starting poller", "source", source, "interval", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Intial poll
	m.poll(ctx, source, url)

	for {
		select {
		case <-ctx.Done():
			slog.Info("poller shutting down", "source", source)
			return
		case <-ticker.C:
			m.poll(ctx, source, url)
		}
	}
}

func (m *Manager) poll(ctx context.Context, source, url string) {
	slog.Debug("polling", "source", source)

	var (
		disasters []*models.Disaster
		err       error
	)

	for attempt := 0; attempt < 5; attempt++ {
		disasters, err = m.pollGDACS(ctx, url)

		if err == nil {
			break
		}

		if attempt < 4 {
			backoff := time.Duration(1<<attempt) * time.Second
			slog.Warn("poll failed, retrying", "source", source, "attempt", attempt+1, "backoff", backoff, "error", err)

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}

	if err != nil {
		slog.Error("poll failed after 5 attempts", "source", source, "error", err)
		return
	}

	// Filter out existing disasters before submitting
	var newDisasters []*models.Disaster
	for _, d := range disasters {
		exists, err := m.repo.Exists(ctx, d.ID)
		if err != nil {
			slog.Error("error checking existence", "id", d.ID, "error", err)
			continue
		}
		if !exists {
			newDisasters = append(newDisasters, d)
		}
	}

	if len(newDisasters) == 0 {
		slog.Info("no new disaster alerts found", "source", source)
		return
	}

	for _, d := range newDisasters {
		m.pool.Submit(d)
	}

	slog.Debug("poll complete", "source", source, "count", len(newDisasters))
}

func (m *Manager) Stop() {
	m.wg.Wait()
	m.pool.Stop()
	slog.Info("ingestion manager stopped")
}

