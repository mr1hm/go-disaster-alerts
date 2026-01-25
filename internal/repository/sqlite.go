package repository

import (
	"database/sql"
	"fmt"

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

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}
