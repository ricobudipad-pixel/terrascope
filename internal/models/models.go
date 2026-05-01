package models

import "time"

type Scan struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	ConfigType   string    `json:"config_type"` // terraform, kubernetes, docker-compose, cloudformation
	Status       string    `json:"status"`      // pending, scanning, complete, failed
	TotalDrifts  int       `json:"total_drifts"`
	Critical     int       `json:"critical"`
	High         int       `json:"high"`
	Medium       int       `json:"medium"`
	Low          int       `json:"low"`
	TokensUsed   int       `json:"tokens_used"`
	ScanTimeMs   int       `json:"scan_time_ms"`
	CreatedAt    time.Time `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

type Drift struct {
	ID          int    `json:"id"`
	ScanID      int    `json:"scan_id"`
	Resource    string `json:"resource"`
	DriftType   string `json:"drift_type"` // missing, modified, added, security, cost
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CurrentValue string `json:"current_value"`
	ExpectedValue string `json:"expected_value"`
	Remediation string `json:"remediation"`
}

type Baseline struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	ConfigType  string    `json:"config_type"`
	Content     string    `json:"content"`
	ResourceCount int     `json:"resource_count"`
	CreatedAt   time.Time `json:"created_at"`
}
