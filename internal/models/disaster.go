package models

import (
	"time"

	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
)

type Disaster struct {
	ID          string // Unique ID from source (e.g., "gdacs_12345")
	Source      string // "GDACS"
	Type        disastersv1.DisasterType
	Title       string
	Description string
	Magnitude   float64 // Richter scale for earthquakes
	AlertLevel  disastersv1.AlertLevel
	Latitude    float64
	Longitude   float64
	Timestamp   time.Time // when the event occurred
	Country     string    // Country where disaster occurred
	Population  string    // Affected population (e.g., "1 thousand (in MMI>=VII)")
	ReportURL   string    // Link to detailed report
	Raw         []byte    // original JSON/XML for debugging
	CreatedAt   time.Time // when we ingested it
}

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func (d *Disaster) Coordinates() Coordinates {
	return Coordinates{
		Latitude:  d.Latitude,
		Longitude: d.Longitude,
	}
}
