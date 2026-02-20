package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
	internalgrpc "github.com/mr1hm/go-disaster-alerts/internal/grpc"
	"github.com/mr1hm/go-disaster-alerts/internal/models"
	"github.com/mr1hm/go-disaster-alerts/internal/repository"
)

type Handler struct {
	repo        repository.DisasterRepository
	broadcaster *internalgrpc.Broadcaster
}

func NewHandler(repo repository.DisasterRepository, broadcaster *internalgrpc.Broadcaster) *Handler {
	return &Handler{
		repo:        repo,
		broadcaster: broadcaster,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/disasters", h.getDisasters)
	r.GET("/health", h.health)
	r.POST("/api/debug/test-disaster", h.createTestDisaster)
}

func (h *Handler) getDisasters(c *gin.Context) {
	filter := repository.Filter{
		Limit: 20, // Default to 20 disasters if limit param not supplied
	}

	if t := c.Query("type"); t != "" {
		dt := parseDisasterType(t)
		if dt != disastersv1.DisasterType_UNSPECIFIED {
			filter.Type = &dt
		}
	}
	if m := c.Query("min_magnitude"); m != "" {
		if mag, err := strconv.ParseFloat(m, 64); err == nil {
			filter.MinMagnitude = &mag
		}
	}
	if s := c.Query("since"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.Since = &t
		}
	}
	if l := c.Query("limit"); l != "" {
		if lim, err := strconv.Atoi(l); err == nil && lim > 0 && lim <= 500 {
			filter.Limit = lim
		}
	}
	if al := c.Query("alert_level"); al != "" {
		level := parseAlertLevel(al)
		if level != disastersv1.AlertLevel_UNKNOWN {
			filter.AlertLevel = &level
		}
	}
	if mal := c.Query("min_alert_level"); mal != "" {
		level := parseAlertLevel(mal)
		if level != disastersv1.AlertLevel_UNKNOWN {
			filter.MinAlertLevel = &level
		}
	}

	disasters, err := h.repo.ListDisasters(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch disasters",
		})
		return
	}

	fc := toGeoJSON(disasters)
	c.Header("Content-Type", "application/geo+json")
	c.JSON(http.StatusOK, fc)
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) createTestDisaster(c *gin.Context) {
	disaster := &models.Disaster{
		ID:          fmt.Sprintf("test_%d", time.Now().UnixNano()),
		Source:      "TEST",
		Type:        disastersv1.DisasterType_EARTHQUAKE,
		Title:       "Test Earthquake - M7.5",
		Description: "This is a test disaster for debugging",
		Magnitude:   7.5,
		AlertLevel:  disastersv1.AlertLevel_RED,
		Latitude:    35.6762,
		Longitude:   139.6503,
		Timestamp:   time.Now(),
		Country:                 "Japan",
		AffectedPopulation:      "50 thousand (in MMI>=VII)",
		AffectedPopulationCount: 50000,
		ReportURL:               "https://www.gdacs.org/report.aspx?eventtype=EQ&eventid=test",
		CreatedAt:   time.Now(),
	}

	// Broadcast only - don't persist test data to DB
	if h.broadcaster != nil {
		h.broadcaster.Broadcast(disaster)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "test disaster broadcast (not persisted)",
		"id":      disaster.ID,
	})
}

func parseDisasterType(s string) disastersv1.DisasterType {
	switch strings.ToLower(s) {
	case "earthquake":
		return disastersv1.DisasterType_EARTHQUAKE
	case "flood":
		return disastersv1.DisasterType_FLOOD
	case "cyclone":
		return disastersv1.DisasterType_CYCLONE
	case "tsunami":
		return disastersv1.DisasterType_TSUNAMI
	case "volcano":
		return disastersv1.DisasterType_VOLCANO
	case "wildfire":
		return disastersv1.DisasterType_WILDFIRE
	case "drought":
		return disastersv1.DisasterType_DROUGHT
	default:
		return disastersv1.DisasterType_UNSPECIFIED
	}
}

func parseAlertLevel(s string) disastersv1.AlertLevel {
	switch strings.ToLower(s) {
	case "green":
		return disastersv1.AlertLevel_GREEN
	case "orange":
		return disastersv1.AlertLevel_ORANGE
	case "red":
		return disastersv1.AlertLevel_RED
	default:
		return disastersv1.AlertLevel_UNKNOWN
	}
}
