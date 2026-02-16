package repository

import (
	"context"
	"testing"
	"time"

	disastersv1 "github.com/mr1hm/go-disaster-alerts/gen/disasters/v1"
	"github.com/mr1hm/go-disaster-alerts/internal/models"
)

func setupTestDB(t *testing.T) *SQLiteDB {
	db, err := NewSQLiteDB(":memory:")
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	return db
}

func TestSQLiteDB_AddAndGetDisaster(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	disaster := &models.Disaster{
		ID:        "test_123",
		Source:    "test",
		Type:      disastersv1.DisasterType_EARTHQUAKE,
		Title:     "Test Earthquake",
		Magnitude: 5.5,
		Latitude:  35.0,
		Longitude: 139.0,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	// Add
	err := db.Add(ctx, disaster)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Get
	got, err := db.GetByID(ctx, "test_123")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Title != "Test Earthquake" {
		t.Errorf("expected title 'Test Earthquake', got '%s'", got.Title)
	}
}

func TestSQLiteDB_Exists(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	exists, err := db.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("expected false for nonexistent ID")
	}

	// Add one
	db.Add(ctx, &models.Disaster{
		ID:        "exists_test",
		Source:    "test",
		Type:      disastersv1.DisasterType_FLOOD,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	})

	exists, err = db.Exists(ctx, "exists_test")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("expected true for existing ID")
	}
}

func TestSQLiteDB_ListDisasters_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()

	// Add test data
	disasters := []*models.Disaster{
		{ID: "eq1", Source: "test", Type: disastersv1.DisasterType_EARTHQUAKE, Magnitude: 6.0, Timestamp: now, CreatedAt: now},
		{ID: "eq2", Source: "test", Type: disastersv1.DisasterType_EARTHQUAKE, Magnitude: 4.0, Timestamp: now, CreatedAt: now},
		{ID: "fl1", Source: "test", Type: disastersv1.DisasterType_FLOOD, Magnitude: 3.0, Timestamp: now, CreatedAt: now},
	}
	for _, d := range disasters {
		db.Add(ctx, d)
	}

	// Test type filter
	eqType := disastersv1.DisasterType_EARTHQUAKE
	results, err := db.ListDisasters(ctx, Filter{Type: &eqType})
	if err != nil {
		t.Fatalf("ListDisasters failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 earthquakes, got %d", len(results))
	}

	// Test magnitude filter
	minMag := 5.0
	results, err = db.ListDisasters(ctx, Filter{MinMagnitude: &minMag})
	if err != nil {
		t.Fatalf("ListDisasters failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 disaster with mag >= 5.0, got %d", len(results))
	}

	// Test limit
	results, err = db.ListDisasters(ctx, Filter{Limit: 2})
	if err != nil {
		t.Fatalf("ListDisasters failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 disasters with limit, got %d", len(results))
	}
}

func TestSQLiteDB_ListDisasters_AlertLevelFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()

	// Add test data with different alert levels
	disasters := []*models.Disaster{
		{ID: "green1", Source: "test", Type: disastersv1.DisasterType_EARTHQUAKE, AlertLevel: disastersv1.AlertLevel_GREEN, Timestamp: now, CreatedAt: now},
		{ID: "orange1", Source: "test", Type: disastersv1.DisasterType_FLOOD, AlertLevel: disastersv1.AlertLevel_ORANGE, Timestamp: now, CreatedAt: now},
		{ID: "red1", Source: "test", Type: disastersv1.DisasterType_CYCLONE, AlertLevel: disastersv1.AlertLevel_RED, Timestamp: now, CreatedAt: now},
		{ID: "green2", Source: "test", Type: disastersv1.DisasterType_EARTHQUAKE, AlertLevel: disastersv1.AlertLevel_GREEN, Timestamp: now, CreatedAt: now},
	}
	for _, d := range disasters {
		db.Add(ctx, d)
	}

	// Test exact AlertLevel filter
	orange := disastersv1.AlertLevel_ORANGE
	results, err := db.ListDisasters(ctx, Filter{AlertLevel: &orange})
	if err != nil {
		t.Fatalf("ListDisasters failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 orange alert, got %d", len(results))
	}

	// Test MinAlertLevel filter (>= ORANGE should return ORANGE and RED)
	results, err = db.ListDisasters(ctx, Filter{MinAlertLevel: &orange})
	if err != nil {
		t.Fatalf("ListDisasters failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 disasters with alert >= ORANGE, got %d", len(results))
	}

	// Test MinAlertLevel RED (should return only RED)
	red := disastersv1.AlertLevel_RED
	results, err = db.ListDisasters(ctx, Filter{MinAlertLevel: &red})
	if err != nil {
		t.Fatalf("ListDisasters failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 disaster with alert >= RED, got %d", len(results))
	}
}

func TestSQLiteDB_NewFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	disaster := &models.Disaster{
		ID:         "test_fields",
		Source:     "GDACS",
		Type:       disastersv1.DisasterType_CYCLONE,
		Title:      "Test Cyclone",
		Magnitude:  150.0,
		AlertLevel: disastersv1.AlertLevel_RED,
		Latitude:   -20.0,
		Longitude:  45.0,
		Timestamp:  time.Now(),
		Country:    "Madagascar",
		Population: "1.5 million affected",
		ReportURL:  "https://www.gdacs.org/report.aspx?eventtype=TC&eventid=123",
		CreatedAt:  time.Now(),
	}

	err := db.Add(ctx, disaster)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	got, err := db.GetByID(ctx, "test_fields")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Country != "Madagascar" {
		t.Errorf("expected country 'Madagascar', got '%s'", got.Country)
	}
	if got.Population != "1.5 million affected" {
		t.Errorf("expected population '1.5 million affected', got '%s'", got.Population)
	}
	if got.ReportURL != "https://www.gdacs.org/report.aspx?eventtype=TC&eventid=123" {
		t.Errorf("expected report_url, got '%s'", got.ReportURL)
	}
}

func TestSQLiteDB_MarkAsSent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()

	// Add test disasters
	disasters := []*models.Disaster{
		{ID: "sent1", Source: "test", Type: disastersv1.DisasterType_EARTHQUAKE, Timestamp: now, CreatedAt: now},
		{ID: "sent2", Source: "test", Type: disastersv1.DisasterType_FLOOD, Timestamp: now, CreatedAt: now},
		{ID: "sent3", Source: "test", Type: disastersv1.DisasterType_CYCLONE, Timestamp: now, CreatedAt: now},
	}
	for _, d := range disasters {
		db.Add(ctx, d)
	}

	// Mark two as sent
	count, err := db.MarkAsSent(ctx, []string{"sent1", "sent2"})
	if err != nil {
		t.Fatalf("MarkAsSent failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 rows affected, got %d", count)
	}

	// Marking non-existent IDs should return 0
	count, err = db.MarkAsSent(ctx, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("MarkAsSent failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows affected for non-existent ID, got %d", count)
	}

	// Empty slice should return 0
	count, err = db.MarkAsSent(ctx, []string{})
	if err != nil {
		t.Fatalf("MarkAsSent failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows affected for empty slice, got %d", count)
	}
}

func TestSQLiteDB_DuplicateAdd(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	disaster := &models.Disaster{
		ID:        "dup_test",
		Source:    "test",
		Type:      disastersv1.DisasterType_EARTHQUAKE,
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	// First add should succeed
	err := db.Add(ctx, disaster)
	if err != nil {
		t.Fatalf("First Add failed: %v", err)
	}

	// Second add should fail (duplicate primary key)
	err = db.Add(ctx, disaster)
	if err == nil {
		t.Error("expected error for duplicate ID, got nil")
	}
}
