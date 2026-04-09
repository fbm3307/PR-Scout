// Package github provides a GitHub API client for scanning org PRs.
package github

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	gh "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// Client wraps the go-github client for PR scanning operations.
type Client struct {
	client   *gh.Client
	token    string
	org      string
	username string
	logger   *slog.Logger

	protectionMu    sync.Mutex
	protectionCache map[string]*gh.Protection
}

// NewClient creates a GitHub client authenticated with the given token.
func NewClient(token, org, username string, logger *slog.Logger) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &Client{
		client:   gh.NewClient(tc),
		token:    token,
		org:      org,
		username: username,
		logger:   logger,
	}
}

// ValidateCredentials checks that the token is valid.
func (c *Client) ValidateCredentials(ctx context.Context) error {
	_, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("github authentication failed: %w", err)
	}
	return nil
}

// ListOrgRepos returns all repositories in the configured org.
func (c *Client) ListOrgRepos(ctx context.Context) ([]*gh.Repository, error) {
	var allRepos []*gh.Repository
	opts := &gh.RepositoryListByOrgOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := c.client.Repositories.ListByOrg(ctx, c.org, opts)
		if err != nil {
			return nil, fmt.Errorf("list repos for org %s: %w", c.org, err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	c.logger.Info("Listed org repos", "org", c.org, "count", len(allRepos))
	return allRepos, nil
}

// ListOpenPRs returns all open pull requests for a given repo.
func (c *Client) ListOpenPRs(ctx context.Context, repo string) ([]*gh.PullRequest, error) {
	var allPRs []*gh.PullRequest
	opts := &gh.PullRequestListOptions{
		State:       "open",
		ListOptions: gh.ListOptions{PerPage: 50},
	}

	for {
		prs, resp, err := c.client.PullRequests.List(ctx, c.org, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("list PRs for %s/%s: %w", c.org, repo, err)
		}
		allPRs = append(allPRs, prs...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allPRs, nil
}

// ListRecentlyMergedPRs returns PRs that were merged after `since`.
// It fetches closed PRs sorted by most recently updated and stops
// paginating once it reaches PRs older than `since`.
func (c *Client) ListRecentlyMergedPRs(ctx context.Context, repo string, since time.Time) ([]*gh.PullRequest, error) {
	var merged []*gh.PullRequest
	opts := &gh.PullRequestListOptions{
		State:       "closed",
		Sort:        "updated",
		Direction:   "desc",
		ListOptions: gh.ListOptions{PerPage: 50},
	}

	for {
		prs, resp, err := c.client.PullRequests.List(ctx, c.org, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("list merged PRs for %s/%s: %w", c.org, repo, err)
		}

		done := false
		for _, pr := range prs {
			if pr.GetUpdatedAt().Time.Before(since) {
				done = true
				break
			}
			mergedAt := pr.GetMergedAt()
			if !mergedAt.IsZero() && mergedAt.Time.After(since) {
				merged = append(merged, pr)
			}
		}

		if done || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return merged, nil
}

// GetPRFiles returns the list of files changed in a PR.
func (c *Client) GetPRFiles(ctx context.Context, repo string, prNumber int) ([]*gh.CommitFile, error) {
	var allFiles []*gh.CommitFile
	opts := &gh.ListOptions{PerPage: 100}

	for {
		files, resp, err := c.client.PullRequests.ListFiles(ctx, c.org, repo, prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("list files for %s/%s#%d: %w", c.org, repo, prNumber, err)
		}
		allFiles = append(allFiles, files...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allFiles, nil
}

// ListReviews returns all reviews on a PR.
func (c *Client) ListReviews(ctx context.Context, repo string, prNumber int) ([]*gh.PullRequestReview, error) {
	var allReviews []*gh.PullRequestReview
	opts := &gh.ListOptions{PerPage: 50}

	for {
		reviews, resp, err := c.client.PullRequests.ListReviews(ctx, c.org, repo, prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("list reviews for %s/%s#%d: %w", c.org, repo, prNumber, err)
		}
		allReviews = append(allReviews, reviews...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allReviews, nil
}

// ListReviewComments returns all review comments (inline comments) on a PR.
func (c *Client) ListReviewComments(ctx context.Context, repo string, prNumber int) ([]*gh.PullRequestComment, error) {
	var allComments []*gh.PullRequestComment
	opts := &gh.PullRequestListCommentsOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		comments, resp, err := c.client.PullRequests.ListComments(ctx, c.org, repo, prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("list comments for %s/%s#%d: %w", c.org, repo, prNumber, err)
		}
		allComments = append(allComments, comments...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allComments, nil
}

// ListIssueComments returns all issue-level (non-inline) comments on a PR.
func (c *Client) ListIssueComments(ctx context.Context, repo string, prNumber int) ([]*gh.IssueComment, error) {
	var allComments []*gh.IssueComment
	opts := &gh.IssueListCommentsOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		comments, resp, err := c.client.Issues.ListComments(ctx, c.org, repo, prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("list issue comments for %s/%s#%d: %w", c.org, repo, prNumber, err)
		}
		allComments = append(allComments, comments...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allComments, nil
}

// ListCheckRuns returns all check runs for a given git ref (SHA or branch).
func (c *Client) ListCheckRuns(ctx context.Context, repo, ref string) ([]*gh.CheckRun, error) {
	var allRuns []*gh.CheckRun
	opts := &gh.ListCheckRunsOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	for {
		result, resp, err := c.client.Checks.ListCheckRunsForRef(ctx, c.org, repo, ref, opts)
		if err != nil {
			return nil, fmt.Errorf("list check runs for %s/%s@%s: %w", c.org, repo, ref, err)
		}
		allRuns = append(allRuns, result.CheckRuns...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRuns, nil
}

// GetBranchProtection returns branch protection rules, cached per repo+branch for the
// lifetime of the client (branch protection rarely changes within a scan).
func (c *Client) GetBranchProtection(ctx context.Context, repo, branch string) (*gh.Protection, error) {
	key := repo + ":" + branch

	c.protectionMu.Lock()
	if cached, ok := c.protectionCache[key]; ok {
		c.protectionMu.Unlock()
		return cached, nil
	}
	c.protectionMu.Unlock()

	prot, _, err := c.client.Repositories.GetBranchProtection(ctx, c.org, repo, branch)
	if err != nil {
		return nil, fmt.Errorf("get branch protection for %s/%s:%s: %w", c.org, repo, branch, err)
	}

	c.protectionMu.Lock()
	if c.protectionCache == nil {
		c.protectionCache = make(map[string]*gh.Protection)
	}
	c.protectionCache[key] = prot
	c.protectionMu.Unlock()

	return prot, nil
}

// Username returns the configured GitHub username.
func (c *Client) Username() string {
	return c.username
}
