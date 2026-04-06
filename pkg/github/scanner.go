package github

import (
	"context"
	"log/slog"
	"strings"

	gh "github.com/google/go-github/v68/github"

	"github.com/codeready-toolchain/pr-scout/pkg/models"
)

// ScanResult contains the output of scanning a single PR.
type ScanResult struct {
	PR             models.TrackedPR
	ChangedFiles   []models.ChangedFile
	ReviewComments []models.ReviewComment
	MyReview       *models.MyReviewStatus
}

// Scanner performs org-wide PR scanning using the GitHub API.
type Scanner struct {
	client *Client
	logger *slog.Logger
}

// NewScanner creates a Scanner backed by the given Client.
func NewScanner(client *Client, logger *slog.Logger) *Scanner {
	return &Scanner{client: client, logger: logger}
}

// ScanRepo scans a single repository for open PRs and returns results.
func (s *Scanner) ScanRepo(ctx context.Context, repo string) ([]ScanResult, error) {
	prs, err := s.client.ListOpenPRs(ctx, repo)
	if err != nil {
		return nil, err
	}

	var results []ScanResult
	for _, pr := range prs {
		result, err := s.scanPR(ctx, repo, pr)
		if err != nil {
			s.logger.Warn("Failed to scan PR, skipping",
				"repo", repo, "pr", pr.GetNumber(), "error", err)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

func (s *Scanner) scanPR(ctx context.Context, repo string, pr *gh.PullRequest) (*ScanResult, error) {
	prNumber := pr.GetNumber()

	// Fetch changed files
	ghFiles, err := s.client.GetPRFiles(ctx, repo, prNumber)
	if err != nil {
		s.logger.Warn("Failed to fetch files", "repo", repo, "pr", prNumber, "error", err)
	}

	var changedFiles []models.ChangedFile
	var additions, deletions int
	for _, f := range ghFiles {
		changedFiles = append(changedFiles, models.ChangedFile{
			Filename:  f.GetFilename(),
			Status:    f.GetStatus(),
			Additions: f.GetAdditions(),
			Deletions: f.GetDeletions(),
			Patch:     f.GetPatch(),
		})
		additions += f.GetAdditions()
		deletions += f.GetDeletions()
	}

	// Build labels string
	var labelNames []string
	for _, l := range pr.Labels {
		labelNames = append(labelNames, l.GetName())
	}

	title := pr.GetTitle()
	isDraft := pr.GetDraft() || isWIPTitle(title)

	tracked := models.TrackedPR{
		Repo:              repo,
		PRNumber:          prNumber,
		Title:             title,
		Body:              pr.GetBody(),
		Author:            pr.GetUser().GetLogin(),
		URL:               pr.GetHTMLURL(),
		State:             pr.GetState(),
		HeadBranch:        pr.GetHead().GetRef(),
		BaseBranch:        pr.GetBase().GetRef(),
		CreatedAt:         pr.GetCreatedAt().Time,
		UpdatedAt:         pr.GetUpdatedAt().Time,
		Labels:            strings.Join(labelNames, ","),
		IsDraft:           isDraft,
		ChangedFilesCount: len(changedFiles),
		Additions:         additions,
		Deletions:         deletions,
	}

	// Determine if this is the user's own PR
	isMyPR := strings.EqualFold(tracked.Author, s.client.Username())
	tracked.IsMyPR = isMyPR

	// Fetch review comments and human review summary
	reviewComments, myStatus, humanSummary, reviews := s.fetchReviewInfo(ctx, repo, prNumber, tracked.Author)
	tracked.HumanReviewSummary = humanSummary.ToJSON()

	// Capture CodeRabbit PR-level reviews (not inline threads) as review comments
	// so they are detected by SummarizeCodeRabbitFindings.
	hasCodeRabbit := false
	for _, r := range reviews {
		login := r.GetUser().GetLogin()
		body := r.GetBody()
		if strings.EqualFold(login, codeRabbitBotLogin) && body != "" {
			hasCodeRabbit = true
			reviewComments = append(reviewComments, models.ReviewComment{
				Commenter: login,
				Body:      body,
				CreatedAt: r.GetSubmittedAt().Time,
				IsBot:     true,
				BotName:   "coderabbitai",
			})
		}
	}

	// Also check issue comments for CodeRabbit summary comments (posted when
	// CodeRabbit reviews but has no actionable inline comments).
	if !hasCodeRabbit {
		issueComments, err := s.client.ListIssueComments(ctx, repo, prNumber)
		if err != nil {
			s.logger.Warn("Failed to fetch issue comments", "repo", repo, "pr", prNumber, "error", err)
		} else {
			for _, c := range issueComments {
				login := c.GetUser().GetLogin()
				if strings.EqualFold(login, codeRabbitBotLogin) {
					hasCodeRabbit = true
					reviewComments = append(reviewComments, models.ReviewComment{
						Commenter: login,
						Body:      c.GetBody(),
						CreatedAt: c.GetCreatedAt().Time,
						IsBot:     true,
						BotName:   "coderabbitai",
					})
					break
				}
			}
		}
	}

	// For your own PRs: don't track your review status (you don't review your own PRs).
	// Instead, check review approvals and whether reviewers left comments you haven't responded to.
	if isMyPR {
		s.logger.Info("Building my-PR status", "repo", repo, "pr", prNumber, "reviews_count", len(reviews))
		myStatus = s.buildMyPRStatus(ctx, repo, prNumber, reviewComments, reviews)
		s.logger.Info("My-PR status result", "repo", repo, "pr", prNumber, "status", myStatus.Status, "review_state", myStatus.ReviewState)
	}

	// Fetch CI check run status
	ciStatus := s.fetchCIStatus(ctx, repo, pr.GetHead().GetSHA(), tracked.BaseBranch)
	tracked.CIStatusJSON = ciStatus.ToJSON()

	// Fetch CodeRabbit thread resolution via GraphQL
	crResolution, err := s.client.FetchCodeRabbitResolution(ctx, repo, prNumber)
	if err != nil {
		s.logger.Warn("Failed to fetch CodeRabbit resolution", "repo", repo, "pr", prNumber, "error", err)
	} else {
		tracked.CodeRabbitTotal = crResolution.Total
		tracked.CodeRabbitResolved = crResolution.Resolved
	}

	return &ScanResult{
		PR:             tracked,
		ChangedFiles:   changedFiles,
		ReviewComments: reviewComments,
		MyReview:       myStatus,
	}, nil
}

// buildMyPRStatus checks review approvals and unanswered reviewer comments on your own PR.
func (s *Scanner) buildMyPRStatus(ctx context.Context, repo string, prNumber int, comments []models.ReviewComment, reviews []*gh.PullRequestReview) *models.MyReviewStatus {
	username := s.client.Username()

	// Check actual review states from non-bot, non-author reviewers.
	// Keep the latest state per reviewer (same logic as buildHumanReviewSummary).
	type reviewerInfo struct {
		state       string
		submittedAt int64
	}
	latestReviews := make(map[string]*reviewerInfo)
	for _, r := range reviews {
		login := r.GetUser().GetLogin()
		state := strings.ToLower(r.GetState())
		s.logger.Debug("buildMyPRStatus review entry", "login", login, "state", state, "username", username)
		if login == "" || strings.HasSuffix(login, "[bot]") || strings.EqualFold(login, username) {
			continue
		}
		ts := r.GetSubmittedAt().Unix()
		if prev, ok := latestReviews[login]; !ok || ts > prev.submittedAt {
			latestReviews[login] = &reviewerInfo{state: state, submittedAt: ts}
		}
	}

	hasApproval := false
	hasChangesRequested := false
	for reviewer, ri := range latestReviews {
		s.logger.Info("buildMyPRStatus latest review", "reviewer", reviewer, "state", ri.state)
		switch ri.state {
		case "approved":
			hasApproval = true
		case "changes_requested":
			hasChangesRequested = true
		}
	}

	if hasChangesRequested {
		return &models.MyReviewStatus{
			ReviewState: "author",
			Status:      models.ReviewStatusNeedsAttention,
		}
	}
	if hasApproval {
		return &models.MyReviewStatus{
			ReviewState: "author",
			Status:      models.ReviewStatusApproved,
		}
	}

	// No review approvals or change requests -- fall back to comment-based heuristic
	var othersCommentCount int
	var latestOtherComment *models.ReviewComment
	for i, c := range comments {
		if c.IsBot || strings.EqualFold(c.Commenter, username) {
			continue
		}
		othersCommentCount++
		if latestOtherComment == nil || c.CreatedAt.After(latestOtherComment.CreatedAt) {
			latestOtherComment = &comments[i]
		}
	}

	if othersCommentCount == 0 {
		return &models.MyReviewStatus{
			ReviewState: "author",
			Status:      models.ReviewStatusWaiting,
		}
	}

	var myLatestReply *models.ReviewComment
	for i, c := range comments {
		if !strings.EqualFold(c.Commenter, username) {
			continue
		}
		if myLatestReply == nil || c.CreatedAt.After(myLatestReply.CreatedAt) {
			myLatestReply = &comments[i]
		}
	}

	status := &models.MyReviewStatus{
		ReviewState:        "author",
		UnresolvedComments: othersCommentCount,
	}

	if myLatestReply != nil && latestOtherComment != nil && myLatestReply.CreatedAt.After(latestOtherComment.CreatedAt) {
		status.Status = models.ReviewStatusWaiting
	} else {
		status.Status = models.ReviewStatusNeedsAttention
	}

	return status
}

func (s *Scanner) fetchReviewInfo(ctx context.Context, repo string, prNumber int, prAuthor string) ([]models.ReviewComment, *models.MyReviewStatus, *models.HumanReviewSummary, []*gh.PullRequestReview) {
	var comments []models.ReviewComment

	// Fetch inline review comments
	ghComments, err := s.client.ListReviewComments(ctx, repo, prNumber)
	if err != nil {
		s.logger.Warn("Failed to fetch review comments", "repo", repo, "pr", prNumber, "error", err)
	}

	for _, c := range ghComments {
		commenter := c.GetUser().GetLogin()
		isBot := strings.HasSuffix(commenter, "[bot]")

		var botName string
		if isBot {
			botName = strings.TrimSuffix(commenter, "[bot]")
		}

		comments = append(comments, models.ReviewComment{
			Commenter: commenter,
			Body:      c.GetBody(),
			FilePath:  c.GetPath(),
			Line:      c.GetLine(),
			CreatedAt: c.GetCreatedAt().Time,
			IsBot:     isBot,
			BotName:   botName,
		})
	}

	// Fetch reviews to determine user's review state and build human review summary
	reviews, err := s.client.ListReviews(ctx, repo, prNumber)
	if err != nil {
		s.logger.Warn("Failed to fetch reviews", "repo", repo, "pr", prNumber, "error", err)
		return comments, nil, &models.HumanReviewSummary{}, nil
	}

	myStatus := s.buildMyReviewStatus(reviews)
	humanSummary := buildHumanReviewSummary(reviews, prAuthor)
	return comments, myStatus, humanSummary, reviews
}

// buildHumanReviewSummary aggregates the latest review state from every non-bot reviewer,
// excluding the PR author (whose reply-comments show up as review events).
func buildHumanReviewSummary(reviews []*gh.PullRequestReview, prAuthor string) *models.HumanReviewSummary {
	type reviewerState struct {
		state       string
		submittedAt int64
	}
	latest := make(map[string]*reviewerState)

	for _, r := range reviews {
		login := r.GetUser().GetLogin()
		if login == "" || strings.HasSuffix(login, "[bot]") {
			continue
		}
		if strings.EqualFold(login, prAuthor) {
			continue
		}
		state := strings.ToLower(r.GetState())
		ts := r.GetSubmittedAt().Unix()

		if prev, ok := latest[login]; !ok || ts > prev.submittedAt {
			latest[login] = &reviewerState{state: state, submittedAt: ts}
		}
	}

	summary := &models.HumanReviewSummary{}
	for login, rs := range latest {
		summary.TotalReviewers++
		switch rs.state {
		case "approved":
			summary.ApprovedBy = append(summary.ApprovedBy, login)
		case "changes_requested":
			summary.ChangesRequestedBy = append(summary.ChangesRequestedBy, login)
		default:
			summary.CommentedBy = append(summary.CommentedBy, login)
		}
	}
	return summary
}

func (s *Scanner) buildMyReviewStatus(reviews []*gh.PullRequestReview) *models.MyReviewStatus {
	username := s.client.Username()
	if username == "" {
		return nil
	}

	var myStatus *models.MyReviewStatus
	for _, r := range reviews {
		if !strings.EqualFold(r.GetUser().GetLogin(), username) {
			continue
		}

		state := strings.ToLower(r.GetState())
		submittedAt := r.GetSubmittedAt().Time

		if myStatus == nil {
			myStatus = &models.MyReviewStatus{
				ReviewState:    state,
				LastReviewedAt: &submittedAt,
				Status:         models.ReviewStatusWaiting,
			}
		} else if submittedAt.After(*myStatus.LastReviewedAt) {
			myStatus.ReviewState = state
			myStatus.LastReviewedAt = &submittedAt
		}
	}

	return myStatus
}

// fetchCIStatus gathers check-run data for a PR head SHA and cross-references
// with branch protection rules to determine required-check status.
func (s *Scanner) fetchCIStatus(ctx context.Context, repo, headSHA, baseBranch string) *models.CIStatus {
	ci := &models.CIStatus{}

	runs, err := s.client.ListCheckRuns(ctx, repo, headSHA)
	if err != nil {
		s.logger.Warn("Failed to fetch check runs", "repo", repo, "ref", headSHA, "error", err)
		return ci
	}

	// Deduplicate check runs by name, keeping the latest (highest ID)
	latest := make(map[string]*gh.CheckRun)
	for _, r := range runs {
		name := r.GetName()
		if prev, ok := latest[name]; !ok || r.GetID() > prev.GetID() {
			latest[name] = r
		}
	}

	// Build required-check set from branch protection (best-effort)
	requiredSet := make(map[string]bool)
	prot, err := s.client.GetBranchProtection(ctx, repo, baseBranch)
	if err == nil && prot.GetRequiredStatusChecks() != nil {
		rsc := prot.GetRequiredStatusChecks()
		if rsc.Contexts != nil {
			for _, c := range *rsc.Contexts {
				requiredSet[c] = true
			}
		}
		if rsc.Checks != nil {
			for _, chk := range *rsc.Checks {
				requiredSet[chk.Context] = true
			}
		}
	}

	ci.RequiredTotal = len(requiredSet)
	requiredPassed := 0

	for name, r := range latest {
		ci.TotalChecks++
		status := r.GetStatus()
		conclusion := r.GetConclusion()

		switch {
		case status != "completed":
			ci.Pending++
		case conclusion == "success" || conclusion == "skipped" || conclusion == "neutral":
			ci.Passed++
			if requiredSet[name] {
				requiredPassed++
			}
		default:
			ci.Failed++
			summary := ""
			if r.GetOutput() != nil {
				summary = r.GetOutput().GetSummary()
				if len(summary) > 200 {
					summary = summary[:200] + "..."
				}
			}
			ci.FailedChecks = append(ci.FailedChecks, models.CheckSummary{
				Name:       name,
				Conclusion: conclusion,
				Summary:    summary,
			})
		}
	}

	ci.RequiredPassed = requiredPassed
	ci.RequiredAllGreen = ci.RequiredTotal > 0 && requiredPassed == ci.RequiredTotal

	switch {
	case ci.TotalChecks == 0:
		ci.OverallStatus = "pending"
	case ci.Failed > 0:
		ci.OverallStatus = "failure"
	case ci.Pending > 0:
		ci.OverallStatus = "mixed"
	default:
		ci.OverallStatus = "success"
	}

	return ci
}

func isWIPTitle(title string) bool {
	lower := strings.ToLower(title)
	return strings.HasPrefix(lower, "wip:") ||
		strings.HasPrefix(lower, "wip ") ||
		strings.HasPrefix(lower, "[wip]") ||
		strings.Contains(lower, "do not merge") ||
		strings.Contains(lower, "don't merge") ||
		strings.Contains(lower, "do-not-merge")
}
