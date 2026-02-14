package models

import (
	"time"

	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
)

type Disaster struct {
	ID          string // Unique ID from source (e.g., "usgs_us7000abc")
	Source      string // "USGS" or "GDACS"
	Type        disastersv1.DisasterType
	Title       string
	Description string
	Magnitude   float64 // Richter scale for earthquakes
	AlertLevel  disastersv1.AlertLevel
	Latitude    float64
	Longitude   float64
	Timestamp   time.Time // when the event occurred
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
