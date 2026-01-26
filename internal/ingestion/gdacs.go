package ingestion

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/http"
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
	Title       string  `xml:"title"`
	Description string  `xml:"description"`
	Link        string  `xml:"link"`
	PubDate     string  `xml:"pubDate"`
	Lat         float64 `xml:"http://www.georss.org/georss point>lat"`
	Lon         float64 `xml:"http://www.georss.org/georss point>lon"`
	EventType   string  `xml:"http://www.gdacs.org gdacs>eventtype"`
	AlertLevel  string  `xml:"http://www.gdacs.org gdacs>alertlevel"`
	EventID     string  `xml:"http://www.gdacs.org gdacs>eventid"`
	Severity    float64 `xml:"http://www.gdacs.org gdacs>severity"`
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
		disasterType := mapGDACSEventType(item.EventType)
		timestamp, err := time.Parse(time.RFC1123, item.PubDate)
		if err != nil {
			slog.Warn("GDACS timestamp parsing failed", "id", item.EventID, "error", err.Error())
		}

		d := &models.Disaster{
			ID:          "gdacs_" + item.EventID,
			Source:      "gdacs",
			Type:        disasterType,
			Title:       item.Title,
			Description: item.Description,
			Magnitude:   item.Severity,
			Latitude:    item.Lat,
			Longitude:   item.Lon,
			Timestamp:   timestamp,
			CreatedAt:   time.Now(),
		}
		disasters = append(disasters, d)
	}

	return disasters, nil
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
	default:
		return models.DisasterTypeUnknown
	}
}
