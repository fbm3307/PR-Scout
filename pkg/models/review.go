package models

import "time"

// ReviewComment represents a review comment on a PR.
type ReviewComment struct {
	ID        int64     `json:"id"`
	PRID      int64     `json:"pr_id"`
	Commenter string    `json:"commenter"`
	Body      string    `json:"body"`
	FilePath  string    `json:"file_path,omitempty"`
	Line      int       `json:"line,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	IsBot     bool      `json:"is_bot"`
	BotName   string    `json:"bot_name,omitempty"`
	Resolved  bool      `json:"resolved"`
}

// MyReviewStatus tracks the user's review state on a PR.
type MyReviewStatus struct {
	ID                 int64      `json:"id"`
	PRID               int64      `json:"pr_id"`
	LastReviewedAt     *time.Time `json:"last_reviewed_at,omitempty"`
	ReviewState        string     `json:"review_state"` // pending, commented, approved, changes_requested
	Status             string     `json:"status"`       // needs_attention, waiting, approved, merged, closed
	CommitsAfterReview int        `json:"commits_after_review"`
	UnresolvedComments int        `json:"unresolved_comments"`
}

// Review status constants.
const (
	ReviewStatusNeedsAttention = "needs_attention"
	ReviewStatusWaiting        = "waiting"
	ReviewStatusApproved       = "approved"
	ReviewStatusMerged         = "merged"
	ReviewStatusClosed         = "closed"
)

// Review state constants (GitHub review states).
const (
	ReviewStatePending          = "pending"
	ReviewStateCommented        = "commented"
	ReviewStateApproved         = "approved"
	ReviewStateChangesRequested = "changes_requested"
)
