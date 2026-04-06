package models

import "encoding/json"

// CIStatus captures the overall CI/check-run state of a PR.
type CIStatus struct {
	OverallStatus    string         `json:"overall_status"` // success, failure, pending, mixed
	TotalChecks      int            `json:"total_checks"`
	Passed           int            `json:"passed"`
	Failed           int            `json:"failed"`
	Pending          int            `json:"pending"`
	RequiredAllGreen bool           `json:"required_all_green"`
	RequiredTotal    int            `json:"required_total"`
	RequiredPassed   int            `json:"required_passed"`
	FailedChecks     []CheckSummary `json:"failed_checks,omitempty"`
}

// CheckSummary holds the name and outcome of a single failed check run.
type CheckSummary struct {
	Name       string `json:"name"`
	Conclusion string `json:"conclusion"`
	Summary    string `json:"summary,omitempty"`
}

// MarshalJSON returns CIStatus as a JSON string suitable for TEXT column storage.
func (c *CIStatus) ToJSON() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// HumanReviewSummary aggregates review states from all human (non-bot) reviewers.
type HumanReviewSummary struct {
	TotalReviewers     int      `json:"total_reviewers"`
	ApprovedBy         []string `json:"approved_by"`
	ChangesRequestedBy []string `json:"changes_requested_by"`
	CommentedBy        []string `json:"commented_by"`
}

func (h *HumanReviewSummary) ToJSON() string {
	b, _ := json.Marshal(h)
	return string(b)
}
