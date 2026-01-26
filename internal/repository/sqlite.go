package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/mr1hm/go-disaster-alerts/internal/models"
	_ "modernc.org/sqlite"
)

type SQLiteDB struct {
	db *sql.DB
}

func NewSQLiteDB(path string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	db.Exec("PRAGMA journal_mode=WAL")   // Write-ahead logging
	db.Exec("PRAGMA synchronous=NORMAL") // Balance safety/speed

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error while pinging database: %w", err)
	}

	s := &SQLiteDB{
		db: db,
	}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("error while migrating to database: %w", err)
	}

	return s, nil
}

func (s *SQLiteDB) migrate() error {
	schema := `
		CREATE TABLE IF NOT EXISTS disasters (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			type TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			magnitude REAL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			timestamp DATETIME NOT NULL,
			raw BLOB,
			created_at DATETIME NOT NULL
		);

		CREATE TABLE IF NOT EXISTS alerts (
			id TEXT PRIMARY KEY,
			disaster_id TEXT NOT NULL,
			severity TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (disaster_id) REFERENCES disasters(id)
		);

		CREATE INDEX IF NOT EXISTS idx_disasters_timestamp ON disasters(timestamp);
		CREATE INDEX IF NOT EXISTS idx_disasters_type ON disasters(type);
		CREATE INDEX IF NOT EXISTS idx_alerts_disaster_id ON alerts(disaster_id);
  	`

	_, err := s.db.Exec(schema)
	return err
}

// Disaster methods

func (s *SQLiteDB) Add(ctx context.Context, d *models.Disaster) error {
	query := `
		INSERT INTO disasters (id, source, type, title, description, magnitude, latitude, longitude, timestamp, raw, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		d.ID, d.Source, d.Type, d.Title, d.Description,
		d.Magnitude, d.Latitude, d.Longitude, d.Timestamp, d.Raw, d.CreatedAt,
	)
	return err
}

func (s *SQLiteDB) GetByID(ctx context.Context, id string) (*models.Disaster, error) {
	query := `SELECT id, source, type, title, description, magnitude, latitude, longitude, timestamp, raw, created_at FROM disasters WHERE id = ?`

	var d models.Disaster
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID, &d.Source, &d.Type, &d.Title, &d.Description,
		&d.Magnitude, &d.Latitude, &d.Longitude, &d.Timestamp, &d.Raw, &d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *SQLiteDB) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM disasters WHERE id = ?)`
	err := s.db.QueryRowContext(ctx, query, id).Scan(&exists)
	return exists, err
}

func (s *SQLiteDB) ListDisasters(ctx context.Context, opts Filter) ([]models.Disaster, error) {
	query := `SELECT id, source, type, title, description, magnitude, latitude, longitude, timestamp, raw, created_at FROM disasters`
	var conditions []string
	args := []any{}

	if opts.Type != nil {
		conditions = append(conditions, "type = ?")
		args = append(args, *opts.Type)
	}
	if opts.Since != nil {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, *opts.Since)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += ` ORDER BY timestamp DESC`

	if opts.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var disasters []models.Disaster
	for rows.Next() {
		var d models.Disaster
		if err := rows.Scan(
			&d.ID, &d.Source, &d.Type, &d.Title, &d.Description,
			&d.Magnitude, &d.Latitude, &d.Longitude, &d.Timestamp, &d.Raw, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		disasters = append(disasters, d)
	}

	return disasters, rows.Err()
}

// Alert methods

func (s *SQLiteDB) AddAlert(ctx context.Context, a *models.Alert) error {
	query := `INSERT INTO alerts (id, disaster_id, severity, created_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, a.ID, a.DisasterID, a.Severity, a.CreatedAt)
	return err
}

func (s *SQLiteDB) GetByDisasterID(ctx context.Context, disasterID string) ([]models.Alert, error) {
	query := `SELECT id, disaster_id, severity, created_at FROM alerts WHERE disaster_id = ?`

	rows, err := s.db.QueryContext(ctx, query, disasterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var a models.Alert
		if err := rows.Scan(&a.ID, &a.DisasterID, &a.Severity, &a.CreatedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}

	return alerts, rows.Err()
}

func (s *SQLiteDB) ListAlerts(ctx context.Context, opts Filter) ([]models.Alert, error) {
	query := `SELECT id, disaster_id, severity, created_at FROM alerts`
	var conditions []string
	args := []any{}

	if opts.Since != nil {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *opts.Since)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += ` ORDER BY created_at DESC`

	if opts.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []models.Alert
	for rows.Next() {
		var a models.Alert
		if err := rows.Scan(&a.ID, &a.DisasterID, &a.Severity, &a.CreatedAt); err != nil {
			return nil, err
		}
		alerts = append(alerts, a)
	}

	return alerts, rows.Err()
}

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}
