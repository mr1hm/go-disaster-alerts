package api

import (
	"strings"

	"github.com/mr1hm/go-disaster-alerts/internal/models"
)

type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}
type Feature struct {
	Type       string         `json:"type"`
	Geometry   Geometry       `json:"geometry"`
	Properties map[string]any `json:"properties"`
}
type Geometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

func toGeoJSON(disasters []models.Disaster) FeatureCollection {
	features := make([]Feature, 0, len(disasters))

	for _, d := range disasters {
		f := Feature{
			Type: "Feature",
			Geometry: Geometry{
				Type:        "Point",
				Coordinates: []float64{d.Longitude, d.Latitude},
			},
			Properties: map[string]any{
				"id":          d.ID,
				"type":        strings.ToLower(d.Type.String()),
				"title":       d.Title,
				"description": d.Description,
				"magnitude":   d.Magnitude,
				"alert_level": strings.ToLower(d.AlertLevel.String()),
				"source":      d.Source,
				"timestamp":   d.Timestamp,
			},
		}
		features = append(features, f)
	}

	return FeatureCollection{
		Type:     "FeatureCollection",
		Features: features,
	}
}
