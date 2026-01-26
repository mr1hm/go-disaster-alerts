package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mr1hm/go-disaster-alerts/internal/models"
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
		Limit: 20,
	}

	if t := c.Query("type"); t != "" {
		dt := models.DisasterType(t)
		filter.Type = &dt
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
