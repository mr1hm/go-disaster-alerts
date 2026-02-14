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
