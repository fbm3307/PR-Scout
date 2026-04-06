package github

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/codeready-toolchain/pr-scout/pkg/models"
)

// Tracker monitors the user's review comment lifecycle on PRs.
type Tracker struct {
	client *Client
	logger *slog.Logger
}

// NewTracker creates a Tracker for following up on the user's reviews.
func NewTracker(client *Client, logger *slog.Logger) *Tracker {
	return &Tracker{client: client, logger: logger}
}

// ComputeReviewStatus determines whether a PR needs the user's attention
// based on their latest review and subsequent PR activity.
func (t *Tracker) ComputeReviewStatus(
	ctx context.Context,
	repo string,
	prNumber int,
	prState string,
	prUpdatedAt time.Time,
	myStatus *models.MyReviewStatus,
) *models.MyReviewStatus {
	if myStatus == nil {
		return nil
	}

	// Handle terminal PR states
	switch prState {
	case "merged":
		myStatus.Status = models.ReviewStatusMerged
		return myStatus
	case "closed":
		myStatus.Status = models.ReviewStatusClosed
		return myStatus
	}

	// For the PR author, buildMyPRStatus already computed the correct status
	// from actual review approvals -- preserve it.
	if myStatus.ReviewState == "author" {
		return myStatus
	}

	// If user approved, mark as approved
	if myStatus.ReviewState == models.ReviewStateApproved {
		myStatus.Status = models.ReviewStatusApproved
		return myStatus
	}

	// Check if there are new commits since the user's last review
	if myStatus.LastReviewedAt != nil && prUpdatedAt.After(*myStatus.LastReviewedAt) {
		myStatus.Status = models.ReviewStatusNeedsAttention
		myStatus.UnresolvedComments = t.countUnresolvedUserComments(ctx, repo, prNumber)
		return myStatus
	}

	myStatus.Status = models.ReviewStatusWaiting
	return myStatus
}

func (t *Tracker) countUnresolvedUserComments(ctx context.Context, repo string, prNumber int) int {
	comments, err := t.client.ListReviewComments(ctx, repo, prNumber)
	if err != nil {
		t.logger.Warn("Failed to fetch comments for unresolved count",
			"repo", repo, "pr", prNumber, "error", err)
		return 0
	}

	username := t.client.Username()
	count := 0
	for _, c := range comments {
		if strings.EqualFold(c.GetUser().GetLogin(), username) {
			count++
		}
	}
	return count
}
