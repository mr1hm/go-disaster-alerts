package models

import "time"

type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "LOW"
	AlertSeverityModerate AlertSeverity = "MODERATE"
	AlertSeverityHigh     AlertSeverity = "HIGH"
	AlertSeverityCritical AlertSeverity = "CRITICAL"
)

type Alert struct {
	ID         string
	DisasterID string
	Severity   AlertSeverity
	CreatedAt  time.Time
}
