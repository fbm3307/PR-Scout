// Package services provides domain services for pr-scout.
package services

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/codeready-toolchain/pr-scout/pkg/github"
	"github.com/codeready-toolchain/pr-scout/pkg/llm"
	"github.com/codeready-toolchain/pr-scout/pkg/models"
)

// ScanService orchestrates the full PR scanning pipeline.
type ScanService struct {
	db           *sql.DB
	ghClient     *github.Client
	scanner      *github.Scanner
	tracker      *github.Tracker
	llmClient    *llm.Client
	logger       *slog.Logger
	repos        []string
	maxPRAgeDays int

	// Background LLM worker
	llmCancel context.CancelFunc
	llmWG     sync.WaitGroup
}

// NewScanService creates a new scan orchestrator.
func NewScanService(db *sql.DB, ghClient *github.Client, scanner *github.Scanner, tracker *github.Tracker, llmClient *llm.Client, logger *slog.Logger, repos []string, _ int, maxPRAgeDays int) *ScanService {
	return &ScanService{
		db:           db,
		ghClient:     ghClient,
		scanner:      scanner,
		tracker:      tracker,
		llmClient:    llmClient,
		logger:       logger,
		repos:        repos,
		maxPRAgeDays: maxPRAgeDays,
	}
}

// StartLLMWorker starts the background goroutine that processes pending LLM analyses.
func (s *ScanService) StartLLMWorker(ctx context.Context) {
	if s.llmClient == nil {
		s.logger.Info("LLM worker not started (LLM disabled)")
		return
	}

	workerCtx, cancel := context.WithCancel(ctx)
	s.llmCancel = cancel

	s.llmWG.Add(1)
	go func() {
		defer s.llmWG.Done()
		s.logger.Info("LLM background worker started")
		s.llmWorkerLoop(workerCtx)
		s.logger.Info("LLM background worker stopped")
	}()
}

// StopLLMWorker stops the background LLM worker gracefully.
func (s *ScanService) StopLLMWorker() {
	if s.llmCancel != nil {
		s.llmCancel()
	}
	s.llmWG.Wait()
}

// llmWorkerLoop polls for pending PRs and analyzes them one at a time.
func (s *ScanService) llmWorkerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pr, err := s.nextPendingPR()
		if err != nil {
			s.logger.Error("LLM worker: failed to fetch pending PR", "error", err)
			sleepCtx(ctx, 5*time.Second)
			continue
		}

		if pr == nil {
			sleepCtx(ctx, 10*time.Second)
			continue
		}

		s.logger.Info("LLM analyzing PR", "repo", pr.Repo, "pr", pr.PRNumber, "title", pr.Title)

		// Fetch changed files from GitHub for diff context
		ghFiles, err := s.ghClient.GetPRFiles(ctx, pr.Repo, pr.PRNumber)
		if err != nil {
			s.logger.Warn("LLM worker: failed to fetch files", "repo", pr.Repo, "pr", pr.PRNumber, "error", err)
		}
		var files []models.ChangedFile
		for _, f := range ghFiles {
			files = append(files, models.ChangedFile{
				Filename:  f.GetFilename(),
				Status:    f.GetStatus(),
				Additions: f.GetAdditions(),
				Deletions: f.GetDeletions(),
				Patch:     f.GetPatch(),
			})
		}

		analysis, err := s.llmClient.AnalyzePR(ctx, *pr, files)
		if err != nil {
			s.logger.Warn("LLM analysis failed", "repo", pr.Repo, "pr", pr.PRNumber, "error", err)
			// Don't mark as failed permanently -- leave as pending for retry next cycle
			sleepCtx(ctx, 2*time.Second)
			continue
		}

		_, err = s.db.Exec(
			`UPDATE tracked_prs SET ai_summary = ?, review_hints = ?, risk_notes = ?, llm_status = 'completed' WHERE id = ?`,
			analysis.Summary, analysis.ReviewHints, analysis.RiskNotes, pr.ID,
		)
		if err != nil {
			s.logger.Error("LLM worker: failed to update PR", "id", pr.ID, "error", err)
		} else {
			s.logger.Info("LLM analysis completed", "repo", pr.Repo, "pr", pr.PRNumber)
		}
	}
}

// nextPendingPR fetches the next PR that needs LLM analysis.
func (s *ScanService) nextPendingPR() (*models.TrackedPR, error) {
	var pr models.TrackedPR
	var isNew, isMyPR, isDraft int
	var body, labels sql.NullString

	err := s.db.QueryRow(
		`SELECT id, scan_id, repo, pr_number, title, body, author, url, state,
		        head_branch, base_branch, created_at, updated_at, labels,
		        is_new, is_my_pr, is_draft, changed_files_count, additions, deletions
		 FROM tracked_prs WHERE llm_status = 'pending' ORDER BY updated_at DESC LIMIT 1`,
	).Scan(
		&pr.ID, &pr.ScanID, &pr.Repo, &pr.PRNumber, &pr.Title, &body, &pr.Author, &pr.URL, &pr.State,
		&pr.HeadBranch, &pr.BaseBranch, &pr.CreatedAt, &pr.UpdatedAt, &labels,
		&isNew, &isMyPR, &isDraft, &pr.ChangedFilesCount, &pr.Additions, &pr.Deletions,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	pr.IsNew = isNew == 1
	pr.IsMyPR = isMyPR == 1
	pr.IsDraft = isDraft == 1
	pr.Body = body.String
	pr.Labels = labels.String

	return &pr, nil
}

// RunScan executes a PR scan across configured repos. LLM runs separately via background worker.
func (s *ScanService) RunScan(ctx context.Context, ghClient *github.Client) (*models.ScanRun, error) {
	scan := &models.ScanRun{
		StartedAt: time.Now().UTC(),
		Status:    models.ScanStatusRunning,
	}

	scanID, err := s.insertScanRun(scan)
	if err != nil {
		return nil, fmt.Errorf("create scan run: %w", err)
	}
	scan.ID = scanID

	// Determine repos to scan
	var repoNames []string
	if len(s.repos) > 0 {
		repoNames = s.repos
		s.logger.Info("Scanning configured repos", "count", len(repoNames))
	} else {
		repos, err := ghClient.ListOrgRepos(ctx)
		if err != nil {
			s.failScan(scan, err)
			return scan, err
		}
		for _, r := range repos {
			repoNames = append(repoNames, r.GetName())
		}
	}
	scan.ReposScanned = len(repoNames)

	// Load previous scan data for change detection
	previousPRs, err := s.loadPreviousScanPRs()
	if err != nil {
		s.logger.Warn("Failed to load previous scan PRs", "error", err)
	}

	// Scan repos in parallel (up to 5 concurrent)
	type repoResult struct {
		results []github.ScanResult
		repo    string
	}
	resultsCh := make(chan repoResult, len(repoNames))
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup

	for _, repoName := range repoNames {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results, err := s.scanner.ScanRepo(ctx, repo)
			if err != nil {
				s.logger.Warn("Failed to scan repo, skipping", "repo", repo, "error", err)
				return
			}
			resultsCh <- repoResult{results: results, repo: repo}
		}(repoName)
	}

	// Close channel when all goroutines finish
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	// Collect and persist results
	for rr := range resultsCh {
		for _, result := range rr.results {
			scan.PRsFound++

			prKey := fmt.Sprintf("%s#%d", result.PR.Repo, result.PR.PRNumber)
			prevData, existed := previousPRs[prKey]
			if !existed {
				result.PR.IsNew = true
				scan.NewPRs++
			}

			result.PR.ScanID = scan.ID
			result.PR.CodeRabbitSummary = github.SummarizeCodeRabbitFindings(result.ReviewComments)

			// Determine LLM status
			result.PR.LLMStatus = s.determineLLMStatus(&result.PR, prevData, existed)

			// Compute review status
			if result.MyReview != nil {
				result.MyReview = s.tracker.ComputeReviewStatus(
					ctx, result.PR.Repo, result.PR.PRNumber,
					result.PR.State, result.PR.UpdatedAt, result.MyReview,
				)
			}

			if err := s.persistResult(result); err != nil {
				s.logger.Warn("Failed to persist PR", "repo", rr.repo, "pr", result.PR.PRNumber, "error", err)
			}
		}
	}

	s.completeScan(scan)
	s.logger.Info("Scan completed",
		"repos", scan.ReposScanned, "prs", scan.PRsFound, "new", scan.NewPRs)

	return scan, nil
}

// previousPRData stores data from a previous scan for change detection.
type previousPRData struct {
	updatedAt time.Time
	aiSummary string
	reviewHints string
	riskNotes string
}

// determineLLMStatus decides whether a PR needs LLM analysis.
func (s *ScanService) determineLLMStatus(pr *models.TrackedPR, prev previousPRData, existed bool) string {
	// Skip bots
	if isBot(pr.Author) {
		return "skipped"
	}

	// Skip drafts and WIP
	if pr.IsDraft {
		return "skipped"
	}

	// Skip old PRs
	if s.maxPRAgeDays > 0 && time.Since(pr.CreatedAt).Hours() > float64(s.maxPRAgeDays*24) {
		return "skipped"
	}

	// If PR existed before and hasn't changed, reuse old summary
	if existed && !prev.updatedAt.Before(pr.UpdatedAt) && prev.aiSummary != "" {
		// Copy previous analysis into current PR (will be persisted)
		pr.AISummary = prev.aiSummary
		pr.ReviewHints = prev.reviewHints
		pr.RiskNotes = prev.riskNotes
		return "completed"
	}

	// Needs analysis
	return "pending"
}

func (s *ScanService) insertScanRun(scan *models.ScanRun) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO scan_runs (started_at, status) VALUES (?, ?)`,
		scan.StartedAt, scan.Status,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *ScanService) completeScan(scan *models.ScanRun) {
	now := time.Now().UTC()
	scan.CompletedAt = &now
	scan.Status = models.ScanStatusCompleted

	_, err := s.db.Exec(
		`UPDATE scan_runs SET completed_at = ?, status = ?, repos_scanned = ?, prs_found = ?, new_prs = ? WHERE id = ?`,
		now, scan.Status, scan.ReposScanned, scan.PRsFound, scan.NewPRs, scan.ID,
	)
	if err != nil {
		s.logger.Error("Failed to update scan run", "id", scan.ID, "error", err)
	}
}

func (s *ScanService) failScan(scan *models.ScanRun, scanErr error) {
	now := time.Now().UTC()
	scan.CompletedAt = &now
	scan.Status = models.ScanStatusFailed
	scan.ErrorMessage = scanErr.Error()

	_, err := s.db.Exec(
		`UPDATE scan_runs SET completed_at = ?, status = ?, error_message = ? WHERE id = ?`,
		now, scan.Status, scan.ErrorMessage, scan.ID,
	)
	if err != nil {
		s.logger.Error("Failed to update scan run as failed", "id", scan.ID, "error", err)
	}
}

// loadPreviousScanPRs returns PR data from the most recent completed scan for change detection.
func (s *ScanService) loadPreviousScanPRs() (map[string]previousPRData, error) {
	result := make(map[string]previousPRData)

	row := s.db.QueryRow(
		`SELECT id FROM scan_runs WHERE status = ? ORDER BY started_at DESC LIMIT 1`,
		models.ScanStatusCompleted,
	)
	var prevScanID int64
	if err := row.Scan(&prevScanID); err != nil {
		return result, nil
	}

	rows, err := s.db.Query(
		`SELECT repo, pr_number, updated_at, ai_summary, review_hints, risk_notes
		 FROM tracked_prs WHERE scan_id = ?`, prevScanID,
	)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		var repo string
		var prNumber int
		var updatedAt time.Time
		var aiSummary, reviewHints, riskNotes sql.NullString
		if err := rows.Scan(&repo, &prNumber, &updatedAt, &aiSummary, &reviewHints, &riskNotes); err != nil {
			continue
		}
		result[fmt.Sprintf("%s#%d", repo, prNumber)] = previousPRData{
			updatedAt:   updatedAt,
			aiSummary:   aiSummary.String,
			reviewHints: reviewHints.String,
			riskNotes:   riskNotes.String,
		}
	}

	return result, rows.Err()
}

func (s *ScanService) persistResult(result github.ScanResult) error {
	res, err := s.db.Exec(
		`INSERT INTO tracked_prs (scan_id, repo, pr_number, title, body, author, url, state,
		 head_branch, base_branch, created_at, updated_at, labels, is_new, is_my_pr, is_draft,
		 llm_status, ai_summary, review_hints, risk_notes, coderabbit_summary,
		 human_review_summary, ci_status, coderabbit_total, coderabbit_resolved,
		 changed_files_count, additions, deletions)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.PR.ScanID, result.PR.Repo, result.PR.PRNumber,
		result.PR.Title, result.PR.Body, result.PR.Author, result.PR.URL, result.PR.State,
		result.PR.HeadBranch, result.PR.BaseBranch, result.PR.CreatedAt, result.PR.UpdatedAt,
		result.PR.Labels, boolToInt(result.PR.IsNew), boolToInt(result.PR.IsMyPR), boolToInt(result.PR.IsDraft),
		result.PR.LLMStatus, result.PR.AISummary, result.PR.ReviewHints, result.PR.RiskNotes,
		result.PR.CodeRabbitSummary,
		result.PR.HumanReviewSummary, result.PR.CIStatusJSON, result.PR.CodeRabbitTotal, result.PR.CodeRabbitResolved,
		result.PR.ChangedFilesCount, result.PR.Additions, result.PR.Deletions,
	)
	if err != nil {
		return fmt.Errorf("insert tracked PR: %w", err)
	}

	prID, _ := res.LastInsertId()

	for _, c := range result.ReviewComments {
		_, err := s.db.Exec(
			`INSERT INTO review_comments (pr_id, commenter, body, file_path, line, created_at, is_bot, bot_name, resolved)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			prID, c.Commenter, c.Body, c.FilePath, c.Line, c.CreatedAt,
			boolToInt(c.IsBot), c.BotName, boolToInt(c.Resolved),
		)
		if err != nil {
			s.logger.Warn("Failed to insert review comment", "pr_id", prID, "error", err)
		}
	}

	if result.MyReview != nil {
		_, err := s.db.Exec(
			`INSERT INTO my_review_status (pr_id, last_reviewed_at, review_state, status, commits_after_review, unresolved_comments)
			 VALUES (?, ?, ?, ?, ?, ?)
			 ON CONFLICT(pr_id) DO UPDATE SET
			   last_reviewed_at = excluded.last_reviewed_at,
			   review_state = excluded.review_state,
			   status = excluded.status,
			   commits_after_review = excluded.commits_after_review,
			   unresolved_comments = excluded.unresolved_comments`,
			prID, result.MyReview.LastReviewedAt, result.MyReview.ReviewState,
			result.MyReview.Status, result.MyReview.CommitsAfterReview, result.MyReview.UnresolvedComments,
		)
		if err != nil {
			s.logger.Warn("Failed to upsert review status", "pr_id", prID, "error", err)
		}
	}

	return nil
}

func isBot(author string) bool {
	if strings.HasSuffix(author, "[bot]") {
		return true
	}
	botNames := []string{"dependabot", "renovate", "github-actions", "codecov", "snyk-bot"}
	lower := strings.ToLower(author)
	for _, bot := range botNames {
		if lower == bot {
			return true
		}
	}
	return false
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func sleepCtx(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
