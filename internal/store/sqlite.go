package store

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ricobudipad-pixel/terrascope/internal/models"
)

type DB struct {
	conn *sql.DB
}

func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS scans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			config_type TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			total_drifts INTEGER DEFAULT 0,
			critical INTEGER DEFAULT 0,
			high INTEGER DEFAULT 0,
			medium INTEGER DEFAULT 0,
			low INTEGER DEFAULT 0,
			tokens_used INTEGER DEFAULT 0,
			scan_time_ms INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS drifts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scan_id INTEGER NOT NULL,
			resource TEXT NOT NULL,
			drift_type TEXT NOT NULL,
			severity TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			current_value TEXT DEFAULT '',
			expected_value TEXT DEFAULT '',
			remediation TEXT DEFAULT '',
			FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS baselines (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			config_type TEXT NOT NULL,
			content TEXT NOT NULL,
			resource_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, q := range queries {
		if _, err := db.conn.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) CreateScan(name, configType string) (int64, error) {
	res, err := db.conn.Exec("INSERT INTO scans (name, config_type, status) VALUES (?, ?, 'pending')", name, configType)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) CompleteScan(id int64, drifts []models.Drift, tokensUsed, scanTimeMs int) error {
	severities := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0}
	for _, d := range drifts {
		severities[d.Severity]++
		if _, err := db.conn.Exec(
			"INSERT INTO drifts (scan_id, resource, drift_type, severity, title, description, current_value, expected_value, remediation) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			id, d.Resource, d.DriftType, d.Severity, d.Title, d.Description, d.CurrentValue, d.ExpectedValue, d.Remediation,
		); err != nil {
			return err
		}
	}
	_, err := db.conn.Exec(
		"UPDATE scans SET status='complete', total_drifts=?, critical=?, high=?, medium=?, low=?, tokens_used=?, scan_time_ms=?, completed_at=? WHERE id=?",
		len(drifts), severities["critical"], severities["high"], severities["medium"], severities["low"], tokensUsed, scanTimeMs, time.Now(), id,
	)
	return err
}

func (db *DB) GetScans(limit int) ([]models.Scan, error) {
	rows, err := db.conn.Query("SELECT id, name, config_type, status, total_drifts, critical, high, medium, low, tokens_used, scan_time_ms, created_at, completed_at FROM scans ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []models.Scan
	for rows.Next() {
		var s models.Scan
		var completed sql.NullTime
		if err := rows.Scan(&s.ID, &s.Name, &s.ConfigType, &s.Status, &s.TotalDrifts, &s.Critical, &s.High, &s.Medium, &s.Low, &s.TokensUsed, &s.ScanTimeMs, &s.CreatedAt, &completed); err != nil {
			return nil, err
		}
		if completed.Valid {
			s.CompletedAt = &completed.Time
		}
		scans = append(scans, s)
	}
	return scans, nil
}

func (db *DB) GetScan(id int) (*models.Scan, []models.Drift, error) {
	var s models.Scan
	var completed sql.NullTime
	err := db.conn.QueryRow("SELECT id, name, config_type, status, total_drifts, critical, high, medium, low, tokens_used, scan_time_ms, created_at, completed_at FROM scans WHERE id=?", id).
		Scan(&s.ID, &s.Name, &s.ConfigType, &s.Status, &s.TotalDrifts, &s.Critical, &s.High, &s.Medium, &s.Low, &s.TokensUsed, &s.ScanTimeMs, &s.CreatedAt, &completed)
	if err != nil {
		return nil, nil, err
	}
	if completed.Valid {
		s.CompletedAt = &completed.Time
	}

	rows, err := db.conn.Query("SELECT id, scan_id, resource, drift_type, severity, title, description, current_value, expected_value, remediation FROM drifts WHERE scan_id=?", id)
	if err != nil {
		return &s, nil, err
	}
	defer rows.Close()

	var drifts []models.Drift
	for rows.Next() {
		var d models.Drift
		if err := rows.Scan(&d.ID, &d.ScanID, &d.Resource, &d.DriftType, &d.Severity, &d.Title, &d.Description, &d.CurrentValue, &d.ExpectedValue, &d.Remediation); err != nil {
			return &s, nil, err
		}
		drifts = append(drifts, d)
	}
	return &s, drifts, nil
}

func (db *DB) SaveBaseline(name, configType, content string, resourceCount int) (int64, error) {
	res, err := db.conn.Exec("INSERT INTO baselines (name, config_type, content, resource_count) VALUES (?, ?, ?, ?)", name, configType, content, resourceCount)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) GetBaselines() ([]models.Baseline, error) {
	rows, err := db.conn.Query("SELECT id, name, config_type, content, resource_count, created_at FROM baselines ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var baselines []models.Baseline
	for rows.Next() {
		var b models.Baseline
		if err := rows.Scan(&b.ID, &b.Name, &b.ConfigType, &b.Content, &b.ResourceCount, &b.CreatedAt); err != nil {
			return nil, err
		}
		baselines = append(baselines, b)
	}
	return baselines, nil
}
