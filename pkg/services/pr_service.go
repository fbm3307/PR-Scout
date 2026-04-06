package services

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/codeready-toolchain/pr-scout/pkg/models"
)

// PRService provides read operations for tracked PRs.
type PRService struct {
	db *sql.DB
}

// NewPRService creates a PR query service.
func NewPRService(db *sql.DB) *PRService {
	return &PRService{db: db}
}

// PRWithReview combines a tracked PR with its review status for API responses.
type PRWithReview struct {
	models.TrackedPR
	MyReview *models.MyReviewStatus `json:"my_review,omitempty"`
}

// ListPRs returns tracked PRs from the latest scan, with optional filters.
func (s *PRService) ListPRs(filter models.PRListFilter) ([]PRWithReview, int, error) {
	// Find the latest completed scan
	latestScanID, err := s.latestScanID()
	if err != nil {
		return nil, 0, err
	}
	if latestScanID == 0 {
		return nil, 0, nil
	}

	// Build query with filters
	where := []string{"p.scan_id = ?"}
	args := []any{latestScanID}

	if filter.Repo != "" {
		where = append(where, "p.repo = ?")
		args = append(args, filter.Repo)
	}
	if filter.State != "" {
		where = append(where, "p.state = ?")
		args = append(args, filter.State)
	}
	if filter.IsNew != nil {
		where = append(where, "p.is_new = ?")
		if *filter.IsNew {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if filter.Author != "" {
		where = append(where, "p.author = ?")
		args = append(args, filter.Author)
	}
	if filter.MyReviewStatus != "" {
		switch filter.MyReviewStatus {
		case "not_reviewed":
			where = append(where, "m.status IS NULL")
		default:
			where = append(where, "m.status = ?")
			args = append(args, filter.MyReviewStatus)
		}
	}
	if filter.CIStatus != "" {
		switch filter.CIStatus {
		case "success":
			where = append(where, "json_extract(p.ci_status, '$.overall_status') = 'success'")
		case "failure":
			where = append(where, "json_extract(p.ci_status, '$.overall_status') = 'failure'")
		case "pending":
			where = append(where, "json_extract(p.ci_status, '$.overall_status') IN ('pending', 'mixed')")
		}
	}
	if filter.CodeRabbitStatus != "" {
		switch filter.CodeRabbitStatus {
		case "reviewed":
			where = append(where, "(p.coderabbit_total > 0 OR (p.coderabbit_summary IS NOT NULL AND p.coderabbit_summary != ''))")
		case "all_resolved":
			where = append(where, "p.coderabbit_total > 0 AND p.coderabbit_total = p.coderabbit_resolved")
		case "has_unresolved":
			where = append(where, "p.coderabbit_total > 0 AND p.coderabbit_resolved < p.coderabbit_total")
		case "no_review":
			where = append(where, "p.coderabbit_total = 0 AND (p.coderabbit_summary IS NULL OR p.coderabbit_summary = '')")
		}
	}

	whereClause := strings.Join(where, " AND ")

	// Count total matching rows
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM tracked_prs p LEFT JOIN my_review_status m ON m.pr_id = p.id WHERE %s`,
		whereClause,
	)
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count PRs: %w", err)
	}

	// Paginate
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 500 {
		perPage = 25
	}
	offset := (page - 1) * perPage

	query := fmt.Sprintf(
		`SELECT p.id, p.scan_id, p.repo, p.pr_number, p.title, p.body, p.author, p.url, p.state,
		        p.head_branch, p.base_branch, p.created_at, p.updated_at, p.labels,
		        p.is_new, p.is_my_pr, p.is_draft, p.llm_status, p.ai_summary, p.review_hints, p.risk_notes, p.coderabbit_summary,
		        p.human_review_summary, p.ci_status, p.coderabbit_total, p.coderabbit_resolved,
		        p.changed_files_count, p.additions, p.deletions,
		        m.id, m.last_reviewed_at, m.review_state, m.status, m.commits_after_review, m.unresolved_comments
		 FROM tracked_prs p
		 LEFT JOIN my_review_status m ON m.pr_id = p.id
		 WHERE %s
		 ORDER BY CASE WHEN p.created_at < datetime('now', '-90 days') THEN 1 ELSE 0 END,
		          p.created_at DESC
		 LIMIT ? OFFSET ?`, whereClause,
	)
	args = append(args, perPage, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list PRs: %w", err)
	}
	defer rows.Close()

	var results []PRWithReview
	for rows.Next() {
		var pr PRWithReview
		var isNew, isMyPR, isDraft int
		var llmStatus sql.NullString
		var body, labels, aiSummary, reviewHints, riskNotes, crSummary sql.NullString
		var humanReviewSummary, ciStatus sql.NullString
		var myID sql.NullInt64
		var myReviewedAt sql.NullTime
		var myReviewState, myStatus sql.NullString
		var myCommitsAfter, myUnresolved sql.NullInt64

		err := rows.Scan(
			&pr.ID, &pr.ScanID, &pr.Repo, &pr.PRNumber, &pr.Title, &body, &pr.Author, &pr.URL, &pr.State,
			&pr.HeadBranch, &pr.BaseBranch, &pr.CreatedAt, &pr.UpdatedAt, &labels,
			&isNew, &isMyPR, &isDraft, &llmStatus, &aiSummary, &reviewHints, &riskNotes, &crSummary,
			&humanReviewSummary, &ciStatus, &pr.CodeRabbitTotal, &pr.CodeRabbitResolved,
			&pr.ChangedFilesCount, &pr.Additions, &pr.Deletions,
			&myID, &myReviewedAt, &myReviewState, &myStatus, &myCommitsAfter, &myUnresolved,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan PR row: %w", err)
		}

		pr.IsNew = isNew == 1
		pr.IsMyPR = isMyPR == 1
		pr.IsDraft = isDraft == 1
		pr.LLMStatus = llmStatus.String
		pr.Body = body.String
		pr.Labels = labels.String
		pr.AISummary = aiSummary.String
		pr.ReviewHints = reviewHints.String
		pr.RiskNotes = riskNotes.String
		pr.CodeRabbitSummary = crSummary.String
		pr.HumanReviewSummary = humanReviewSummary.String
		pr.CIStatusJSON = ciStatus.String

		if myID.Valid {
			review := &models.MyReviewStatus{
				ID:                 myID.Int64,
				PRID:               pr.ID,
				ReviewState:        myReviewState.String,
				Status:             myStatus.String,
				CommitsAfterReview: int(myCommitsAfter.Int64),
				UnresolvedComments: int(myUnresolved.Int64),
			}
			if myReviewedAt.Valid {
				review.LastReviewedAt = &myReviewedAt.Time
			}
			pr.MyReview = review
		}

		results = append(results, pr)
	}

	return results, total, rows.Err()
}

// GetPR returns a single PR by ID, including review comments and review status.
func (s *PRService) GetPR(id int64) (*PRWithReview, []models.ReviewComment, error) {
	pr, err := s.getPRByID(id)
	if err != nil {
		return nil, nil, err
	}
	if pr == nil {
		return nil, nil, nil
	}

	comments, err := s.getReviewComments(id)
	if err != nil {
		return pr, nil, err
	}

	return pr, comments, nil
}

func (s *PRService) getPRByID(id int64) (*PRWithReview, error) {
	var pr PRWithReview
	var isNew, isMyPR, isDraft int
	var llmStatusVal sql.NullString
	var body, labels, aiSummary, reviewHints, riskNotes, crSummary sql.NullString
	var humanReviewSummary, ciStatus sql.NullString
	var myID sql.NullInt64
	var myReviewedAt sql.NullTime
	var myReviewState, myStatus sql.NullString
	var myCommitsAfter, myUnresolved sql.NullInt64

	err := s.db.QueryRow(
		`SELECT p.id, p.scan_id, p.repo, p.pr_number, p.title, p.body, p.author, p.url, p.state,
		        p.head_branch, p.base_branch, p.created_at, p.updated_at, p.labels,
		        p.is_new, p.is_my_pr, p.is_draft, p.llm_status, p.ai_summary, p.review_hints, p.risk_notes, p.coderabbit_summary,
		        p.human_review_summary, p.ci_status, p.coderabbit_total, p.coderabbit_resolved,
		        p.changed_files_count, p.additions, p.deletions,
		        m.id, m.last_reviewed_at, m.review_state, m.status, m.commits_after_review, m.unresolved_comments
		 FROM tracked_prs p
		 LEFT JOIN my_review_status m ON m.pr_id = p.id
		 WHERE p.id = ?`, id,
	).Scan(
		&pr.ID, &pr.ScanID, &pr.Repo, &pr.PRNumber, &pr.Title, &body, &pr.Author, &pr.URL, &pr.State,
		&pr.HeadBranch, &pr.BaseBranch, &pr.CreatedAt, &pr.UpdatedAt, &labels,
		&isNew, &isMyPR, &isDraft, &llmStatusVal, &aiSummary, &reviewHints, &riskNotes, &crSummary,
		&humanReviewSummary, &ciStatus, &pr.CodeRabbitTotal, &pr.CodeRabbitResolved,
		&pr.ChangedFilesCount, &pr.Additions, &pr.Deletions,
		&myID, &myReviewedAt, &myReviewState, &myStatus, &myCommitsAfter, &myUnresolved,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get PR %d: %w", id, err)
	}

	pr.IsNew = isNew == 1
	pr.IsMyPR = isMyPR == 1
	pr.IsDraft = isDraft == 1
	pr.LLMStatus = llmStatusVal.String
	pr.Body = body.String
	pr.Labels = labels.String
	pr.AISummary = aiSummary.String
	pr.ReviewHints = reviewHints.String
	pr.RiskNotes = riskNotes.String
	pr.CodeRabbitSummary = crSummary.String
	pr.HumanReviewSummary = humanReviewSummary.String
	pr.CIStatusJSON = ciStatus.String

	if myID.Valid {
		review := &models.MyReviewStatus{
			ID:                 myID.Int64,
			PRID:               pr.ID,
			ReviewState:        myReviewState.String,
			Status:             myStatus.String,
			CommitsAfterReview: int(myCommitsAfter.Int64),
			UnresolvedComments: int(myUnresolved.Int64),
		}
		if myReviewedAt.Valid {
			review.LastReviewedAt = &myReviewedAt.Time
		}
		pr.MyReview = review
	}

	return &pr, nil
}

func (s *PRService) getReviewComments(prID int64) ([]models.ReviewComment, error) {
	rows, err := s.db.Query(
		`SELECT id, pr_id, commenter, body, file_path, line, created_at, is_bot, bot_name, resolved
		 FROM review_comments WHERE pr_id = ? ORDER BY created_at`, prID,
	)
	if err != nil {
		return nil, fmt.Errorf("get review comments for PR %d: %w", prID, err)
	}
	defer rows.Close()

	var comments []models.ReviewComment
	for rows.Next() {
		var c models.ReviewComment
		var isBot, resolved int
		if err := rows.Scan(&c.ID, &c.PRID, &c.Commenter, &c.Body, &c.FilePath, &c.Line,
			&c.CreatedAt, &isBot, &c.BotName, &resolved); err != nil {
			return nil, fmt.Errorf("scan review comment: %w", err)
		}
		c.IsBot = isBot == 1
		c.Resolved = resolved == 1
		comments = append(comments, c)
	}

	return comments, rows.Err()
}

func (s *PRService) latestScanID() (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM scan_runs WHERE status = ? ORDER BY started_at DESC LIMIT 1`,
		models.ScanStatusCompleted,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}
