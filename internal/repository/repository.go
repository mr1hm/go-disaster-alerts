package repository

import (
	"context"
	"time"

	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
	"github.com/mr1hm/go-disaster-alerts/internal/models"
)

type Filter struct {
	Limit         int
	Offset        int
	Since         *time.Time
	Type          *disastersv1.DisasterType
	MinMagnitude  *float64
	AlertLevel    *disastersv1.AlertLevel
	MinAlertLevel *disastersv1.AlertLevel // >= this level (e.g., ORANGE includes ORANGE and RED)
	DiscordSent   *bool                   // Filter by discord_sent status
}

type DisasterRepository interface {
	Add(ctx context.Context, d *models.Disaster) error
	GetByID(ctx context.Context, id string) (*models.Disaster, error)
	Exists(ctx context.Context, id string) (bool, error)
	ListDisasters(ctx context.Context, opts Filter) ([]models.Disaster, error)
	MarkAsSent(ctx context.Context, ids []string) (int64, error)
}

type AlertRepository interface {
	AddAlert(ctx context.Context, a *models.Alert) error
	GetByDisasterID(ctx context.Context, disasterID string) ([]models.Alert, error)
	ListAlerts(ctx context.Context, opts Filter) ([]models.Alert, error)
}
