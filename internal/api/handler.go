package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
	"github.com/mr1hm/go-disaster-alerts/internal/repository"
)

type Handler struct {
	repo repository.DisasterRepository
}

func NewHandler(repo repository.DisasterRepository) *Handler {
	return &Handler{
		repo: repo,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/disasters", h.getDisasters)
	r.GET("/health", h.health)
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
