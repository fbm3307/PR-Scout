package models

import "time"

// ScanRun represents a single execution of the org-wide PR scan.
type ScanRun struct {
	ID           int64      `json:"id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Status       string     `json:"status"` // running, completed, failed
	ReposScanned int        `json:"repos_scanned"`
	PRsFound     int        `json:"prs_found"`
	NewPRs       int        `json:"new_prs"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// ScanStatus constants.
const (
	ScanStatusRunning   = "running"
	ScanStatusCompleted = "completed"
	ScanStatusFailed    = "failed"
)
