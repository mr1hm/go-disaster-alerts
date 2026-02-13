package models

import "time"

type DisasterType string

const (
	DisasterTypeEarthquake DisasterType = "earthquake"
	DisasterTypeFlood      DisasterType = "flood"
	DisasterTypeCyclone    DisasterType = "cyclone"
	DisasterTypeTsunami    DisasterType = "tsunami"
	DisasterTypeVolcano    DisasterType = "volcano"
	DisasterTypeWildfire   DisasterType = "wildfire"
	DisasterTypeDrought    DisasterType = "drought"
	DisasterTypeUnknown    DisasterType = "unknown"
)

type Disaster struct {
	ID          string // Unique ID from source (e.g., "usgs_us7000abc")
	Source      string // "USGS" or "GDACS"
	Type        DisasterType
	Title       string
	Description string
	Magnitude   float64 // Richter scale for earthquakes, severity for others
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
