package services

import (
	"database/sql"
	"fmt"

	"github.com/codeready-toolchain/pr-scout/pkg/models"
)

// DigestService aggregates stats for the daily digest view.
type DigestService struct {
	db *sql.DB
}

// NewDigestService creates a digest stats service.
func NewDigestService(db *sql.DB) *DigestService {
	return &DigestService{db: db}
}

// Digest contains aggregate stats from the latest scan.
type Digest struct {
	Scan              *models.ScanRun `json:"scan"`
	TotalOpenPRs      int             `json:"total_open_prs"`
	NewPRs            int             `json:"new_prs"`
	NeedsAttention    int             `json:"needs_attention"`
	ReposWithActivity int             `json:"repos_with_activity"`
	TopRepos          []RepoStat      `json:"top_repos"`
}

// RepoStat shows PR counts per repo.
type RepoStat struct {
	Repo    string `json:"repo"`
	PRCount int    `json:"pr_count"`
}

// GetLatestDigest returns the digest for the most recent scan.
func (s *DigestService) GetLatestDigest() (*Digest, error) {
	// Get latest completed scan
	scan, err := s.getLatestScan()
	if err != nil {
		return nil, err
	}
	if scan == nil {
		return &Digest{}, nil
	}

	digest := &Digest{Scan: scan}

	// Total open PRs
	s.db.QueryRow(
		`SELECT COUNT(*) FROM tracked_prs WHERE scan_id = ? AND state = 'open'`, scan.ID,
	).Scan(&digest.TotalOpenPRs)

	// New PRs
	s.db.QueryRow(
		`SELECT COUNT(*) FROM tracked_prs WHERE scan_id = ? AND is_new = 1`, scan.ID,
	).Scan(&digest.NewPRs)

	// Needs attention
	s.db.QueryRow(
		`SELECT COUNT(*) FROM my_review_status m
		 JOIN tracked_prs p ON p.id = m.pr_id
		 WHERE p.scan_id = ? AND m.status = ?`, scan.ID, models.ReviewStatusNeedsAttention,
	).Scan(&digest.NeedsAttention)

	// Repos with activity
	s.db.QueryRow(
		`SELECT COUNT(DISTINCT repo) FROM tracked_prs WHERE scan_id = ?`, scan.ID,
	).Scan(&digest.ReposWithActivity)

	// Top repos by PR count
	rows, err := s.db.Query(
		`SELECT repo, COUNT(*) as cnt FROM tracked_prs WHERE scan_id = ?
		 GROUP BY repo ORDER BY cnt DESC LIMIT 10`, scan.ID,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var stat RepoStat
			if err := rows.Scan(&stat.Repo, &stat.PRCount); err == nil {
				digest.TopRepos = append(digest.TopRepos, stat)
			}
		}
	}

	return digest, nil
}

func (s *DigestService) getLatestScan() (*models.ScanRun, error) {
	var scan models.ScanRun
	var completedAt sql.NullTime
	var errorMessage sql.NullString

	err := s.db.QueryRow(
		`SELECT id, started_at, completed_at, status, repos_scanned, prs_found, new_prs, error_message
		 FROM scan_runs WHERE status = ? ORDER BY started_at DESC LIMIT 1`,
		models.ScanStatusCompleted,
	).Scan(&scan.ID, &scan.StartedAt, &completedAt, &scan.Status,
		&scan.ReposScanned, &scan.PRsFound, &scan.NewPRs, &errorMessage)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest scan: %w", err)
	}

	if completedAt.Valid {
		scan.CompletedAt = &completedAt.Time
	}
	scan.ErrorMessage = errorMessage.String

	return &scan, nil
}
