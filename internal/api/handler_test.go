package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mr1hm/go-disaster-alerts/internal/models"
	"github.com/mr1hm/go-disaster-alerts/internal/repository"
)

// mockRepo implements repository.DisasterRepository for testing
type mockRepo struct {
	disasters []models.Disaster
}

func (m *mockRepo) Add(ctx context.Context, d *models.Disaster) error {
	m.disasters = append(m.disasters, *d)
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*models.Disaster, error) {
	for _, d := range m.disasters {
		if d.ID == id {
			return &d, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) Exists(ctx context.Context, id string) (bool, error) {
	for _, d := range m.disasters {
		if d.ID == id {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepo) ListDisasters(ctx context.Context, opts repository.Filter) ([]models.Disaster, error) {
	results := m.disasters

	// Apply type filter
	if opts.Type != nil {
		var filtered []models.Disaster
		for _, d := range results {
			if d.Type == *opts.Type {
				filtered = append(filtered, d)
			}
		}
		results = filtered
	}

	// Apply magnitude filter
	if opts.MinMagnitude != nil {
		var filtered []models.Disaster
		for _, d := range results {
			if d.Magnitude >= *opts.MinMagnitude {
				filtered = append(filtered, d)
			}
		}
		results = filtered
	}

	// Apply limit
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

func setupTestRouter(repo repository.DisasterRepository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(repo)
	handler.RegisterRoutes(router)
	return router
}

func TestGetDisasters_ReturnsGeoJSON(t *testing.T) {
	repo := &mockRepo{
		disasters: []models.Disaster{
			{
				ID:        "test_1",
				Source:    "test",
				Type:      models.DisasterTypeEarthquake,
				Title:     "Test Quake",
				Magnitude: 5.5,
				Latitude:  35.0,
				Longitude: 139.0,
				Timestamp: time.Now(),
			},
		},
	}

	router := setupTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/disasters", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/geo+json" {
		t.Errorf("expected content-type application/geo+json, got %s", contentType)
	}

	// Parse response
	var fc FeatureCollection
	if err := json.Unmarshal(w.Body.Bytes(), &fc); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if fc.Type != "FeatureCollection" {
		t.Errorf("expected type FeatureCollection, got %s", fc.Type)
	}

	if len(fc.Features) != 1 {
		t.Errorf("expected 1 feature, got %d", len(fc.Features))
	}
}

func TestGetDisasters_TypeFilter(t *testing.T) {
	repo := &mockRepo{
		disasters: []models.Disaster{
			{ID: "eq1", Type: models.DisasterTypeEarthquake, Timestamp: time.Now()},
			{ID: "fl1", Type: models.DisasterTypeFlood, Timestamp: time.Now()},
			{ID: "eq2", Type: models.DisasterTypeEarthquake, Timestamp: time.Now()},
		},
	}

	router := setupTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/disasters?type=earthquake", nil)
	router.ServeHTTP(w, req)

	var fc FeatureCollection
	json.Unmarshal(w.Body.Bytes(), &fc)

	if len(fc.Features) != 2 {
		t.Errorf("expected 2 earthquakes, got %d", len(fc.Features))
	}
}

func TestGetDisasters_MagnitudeFilter(t *testing.T) {
	repo := &mockRepo{
		disasters: []models.Disaster{
			{ID: "d1", Magnitude: 6.0, Timestamp: time.Now()},
			{ID: "d2", Magnitude: 4.0, Timestamp: time.Now()},
			{ID: "d3", Magnitude: 7.5, Timestamp: time.Now()},
		},
	}

	router := setupTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/disasters?min_magnitude=5.0", nil)
	router.ServeHTTP(w, req)

	var fc FeatureCollection
	json.Unmarshal(w.Body.Bytes(), &fc)

	if len(fc.Features) != 2 {
		t.Errorf("expected 2 disasters with mag >= 5.0, got %d", len(fc.Features))
	}
}

func TestGetDisasters_LimitFilter(t *testing.T) {
	repo := &mockRepo{
		disasters: []models.Disaster{
			{ID: "d1", Timestamp: time.Now()},
			{ID: "d2", Timestamp: time.Now()},
			{ID: "d3", Timestamp: time.Now()},
			{ID: "d4", Timestamp: time.Now()},
			{ID: "d5", Timestamp: time.Now()},
		},
	}

	router := setupTestRouter(repo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/disasters?limit=3", nil)
	router.ServeHTTP(w, req)

	var fc FeatureCollection
	json.Unmarshal(w.Body.Bytes(), &fc)

	if len(fc.Features) != 3 {
		t.Errorf("expected 3 disasters, got %d", len(fc.Features))
	}
}

func TestHealth(t *testing.T) {
	router := setupTestRouter(&mockRepo{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}
