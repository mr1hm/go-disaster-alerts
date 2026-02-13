package ingestion

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mr1hm/go-disaster-alerts/internal/models"
)

type gdacsRSS struct {
	Channel gdacsChannel `xml:"channel"`
}
type gdacsChannel struct {
	Items []gdacsItem `xml:"item"`
}
type gdacsItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Point       string `xml:"point"`      // "lat lon" format from georss:point
	EventType   string `xml:"eventtype"`  // gdacs:eventtype
	AlertLevel  string `xml:"alertlevel"` // gdacs:alertlevel
	EventID     string `xml:"eventid"`    // gdacs:eventid
	Severity    string `xml:"severity"`   // gdacs:severity - e.g. "Magnitude 5.6M, Depth:56.4km"
}

func (m *Manager) pollGDACS(ctx context.Context, url string) ([]*models.Disaster, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error doing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d - status: %s", resp.StatusCode, resp.Status)
	}

	var data gdacsRSS
	if err := xml.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("error decoding resp.Body: %w", err)
	}

	disasters := make([]*models.Disaster, 0, len(data.Channel.Items))
	for _, item := range data.Channel.Items {
		// Parse lat/lon from "lat lon" format
		var lat, lon float64
		if parts := strings.Fields(item.Point); len(parts) >= 2 {
			lat, _ = strconv.ParseFloat(parts[0], 64)
			lon, _ = strconv.ParseFloat(parts[1], 64)
		}

		// Skip items without valid ID
		if item.EventID == "" {
			continue
		}

		disasterType := mapGDACSEventType(item.EventType)
		timestamp, err := time.Parse(time.RFC1123, item.PubDate)
		if err != nil {
			slog.Warn("GDACS timestamp parsing failed", "id", item.EventID, "error", err.Error())
		}

		d := &models.Disaster{
			ID:          "gdacs_" + item.EventID,
			Source:      "GDACS",
			Type:        disasterType,
			Title:       item.Title,
			Description: item.Description,
			Magnitude:   parseSeverity(item.Severity),
			Latitude:    lat,
			Longitude:   lon,
			Timestamp:   timestamp,
			CreatedAt:   time.Now(),
		}
		disasters = append(disasters, d)
	}

	return disasters, nil
}

// parseSeverity extracts magnitude from strings like "Magnitude 5.6M, Depth:56.4km"
func parseSeverity(severity string) float64 {
	// Try to find a number after "Magnitude " or just extract first float
	severity = strings.TrimPrefix(severity, "Magnitude ")
	for _, part := range strings.Fields(severity) {
		part = strings.TrimRight(part, "M,")
		if mag, err := strconv.ParseFloat(part, 64); err == nil {
			return mag
		}
	}
	return 0
}

func mapGDACSEventType(eventType string) models.DisasterType {
	switch strings.ToUpper(eventType) {
	case "EQ":
		return models.DisasterTypeEarthquake
	case "TC":
		return models.DisasterTypeCyclone
	case "FL":
		return models.DisasterTypeFlood
	case "VO":
		return models.DisasterTypeVolcano
	case "TS":
		return models.DisasterTypeTsunami
	case "WF":
		return models.DisasterTypeWildfire
	case "DR":
		return models.DisasterTypeDrought
	default:
		return models.DisasterTypeUnknown
	}
}
