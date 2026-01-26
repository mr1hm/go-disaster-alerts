package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mr1hm/go-disaster-alerts/internal/models"
)

type usgsResponse struct {
	Features []usgsFeature `json:"features"`
}

type usgsFeature struct {
	ID         string         `json:"id"`
	Properties usgsProperties `json:"properties"`
	Geometry   usgsGeometry   `json:"geometry"`
}
type usgsProperties struct {
	Mag     float64 `json:"mag"`
	Place   string  `json:"place"`
	Time    int64   `json:"time"` // unix
	Title   string  `json:"title"`
	Tsunami int     `json:"tsunami"` // 0 or 1
}
type usgsGeometry struct {
	Coordinates []float64 `json:"coordinates"` // [lon, lat, depth]
}

func (m *Manager) pollUSGS(ctx context.Context, url string) ([]*models.Disaster, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while doing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d - status: %s", resp.StatusCode, resp.Status)
	}

	var data usgsResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error decoding resp.Body: %w", err)
	}

	disasters := make([]*models.Disaster, 0, len(data.Features))
	for _, f := range data.Features {
		d := &models.Disaster{
			ID:          "usgs_" + f.ID,
			Source:      "usgs",
			Type:        models.DisasterTypeEarthquake,
			Title:       f.Properties.Title,
			Description: f.Properties.Place,
			Magnitude:   f.Properties.Mag,
			Longitude:   f.Geometry.Coordinates[0],
			Latitude:    f.Geometry.Coordinates[1],
			Timestamp:   time.UnixMilli(f.Properties.Time),
			CreatedAt:   time.Now(),
		}
		disasters = append(disasters, d)
	}

	return disasters, nil
}
