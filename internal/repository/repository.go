package repository

import (
	"context"
	"time"

	"github.com/mr1hm/go-disaster-alerts/internal/models"
)

type Filter struct {
	Limit  int
	Offset int
	Since  *time.Time
	Type   *models.DisasterType
}

type DisasterRepository interface {
	Add(ctx context.Context, d *models.Disaster) error
	GetByID(ctx context.Context, id string) (*models.Disaster, error)
	Exists(ctx context.Context, id string) (bool, error)
	List(ctx context.Context, opts Filter) ([]models.Disaster, error)
}

type AlertRepository interface {
	Add(ctx context.Context, a *models.Alert) error
	GetByDisasterID(ctx context.Context, disasterID string) ([]models.Alert, error)
	List(ctx context.Context, opts Filter) ([]models.Alert, error)
}
