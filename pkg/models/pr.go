package models

import "time"

// TrackedPR represents a pull request tracked across scans.
type TrackedPR struct {
	ID        int64  `json:"id"`
	ScanID    int64  `json:"scan_id"`
	Repo      string `json:"repo"`
	PRNumber  int    `json:"pr_number"`
	Title     string `json:"title"`
	Body      string `json:"body,omitempty"`
	Author    string `json:"author"`
	URL       string `json:"url"`
	State     string `json:"state"` // open, closed, merged

	HeadBranch string `json:"head_branch"`
	BaseBranch string `json:"base_branch"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Labels string `json:"labels,omitempty"` // JSON array stored as text

	IsNew       bool   `json:"is_new"`
	IsMyPR      bool   `json:"is_my_pr"`
	IsDraft     bool   `json:"is_draft"`
	LLMStatus   string `json:"llm_status"` // pending, completed, skipped
	AISummary   string `json:"ai_summary,omitempty"`
	ReviewHints string `json:"review_hints,omitempty"`
	RiskNotes   string `json:"risk_notes,omitempty"`

	CodeRabbitSummary string `json:"coderabbit_summary,omitempty"` // JSON stored as text

	HumanReviewSummary string `json:"human_review_summary,omitempty"` // JSON stored as text
	CIStatusJSON       string `json:"ci_status,omitempty"`            // JSON stored as text
	CodeRabbitTotal    int    `json:"coderabbit_total"`
	CodeRabbitResolved int    `json:"coderabbit_resolved"`

	ChangedFilesCount int `json:"changed_files_count"`
	Additions         int `json:"additions"`
	Deletions         int `json:"deletions"`
}

// ChangedFile represents a file modified in a PR.
type ChangedFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"` // added, removed, modified, renamed
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch,omitempty"`
}

// PRListFilter holds query parameters for listing PRs.
type PRListFilter struct {
	Repo             string `json:"repo,omitempty"`
	State            string `json:"state,omitempty"`
	IsNew            *bool  `json:"is_new,omitempty"`
	Author           string `json:"author,omitempty"`
	MyReviewStatus   string `json:"my_review_status,omitempty"`
	CIStatus         string `json:"ci_status,omitempty"`          // success, failure, pending
	CodeRabbitStatus string `json:"coderabbit_status,omitempty"` // all_resolved, has_unresolved
	Page             int    `json:"page"`
	PerPage          int    `json:"per_page"`
}
