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

		exists, err := m.repo.Exists(ctx, disaster.ID)
		if err != nil {
			slog.Error("error checking existence", "id", disaster.ID, "error", err)
			return err
		}
		if exists {
			return nil
		}

		if err := m.repo.Add(ctx, disaster); err != nil {
			slog.Error("error adding disaster", "id", disaster.ID, "error", err)
			return err
		}

		// Broadcast to gRPC stream subscribers if disaster meets criteria
		if m.broadcaster != nil && shouldBroadcast(disaster) {
			m.broadcaster.Broadcast(disaster)
		}

		slog.Info("added disaster", "id", disaster.ID, "type", disaster.Type, "source", disaster.Source)
		return nil
	}

	m.pool = worker.NewWorkerPool(m.cfg.Worker.Count, m.cfg.Worker.BufferSize, processor)
	m.pool.Start(ctx)

	// Start USGS poller if enabled
	if m.cfg.Sources.USGSEnabled {
		m.wg.Add(1)
		go m.runPoller(ctx, "usgs", m.cfg.Sources.USGSURL, m.cfg.Sources.USGSPollInterval)
	}

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

	switch source {
	case "usgs":
		disasters, err = m.pollUSGS(ctx, url)
	case "gdacs":
		disasters, err = m.pollGDACS(ctx, url)
	}
	if err != nil {
		slog.Error("poll failed", "source", source, "error", err)
		return
	}

	for _, d := range disasters {
		m.pool.Submit(d)
	}

	slog.Debug("poll complete", "source", source, "count", len(disasters))
}

func (m *Manager) Stop() {
	m.wg.Wait()
	m.pool.Stop()
	slog.Info("ingestion manager stopped")
}

// shouldBroadcast returns true if disaster meets streaming criteria:
// - Earthquakes: magnitude >= 5.0
// - Other disasters: alert_level is "orange" or "red"
func shouldBroadcast(d *models.Disaster) bool {
	if d.Type == models.DisasterTypeEarthquake {
		return d.Magnitude >= 5.0
	}
	return d.AlertLevel == "orange" || d.AlertLevel == "red"
}
