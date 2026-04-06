package llm

import (
	"fmt"
	"strings"

	"github.com/codeready-toolchain/pr-scout/pkg/models"
)

const reviewerSystemPrompt = `You are a senior code reviewer helping a developer prioritize and focus their PR reviews. Be concise and actionable.`

const authorSystemPrompt = `You are a helpful assistant for a developer who authored this PR. Help them understand if there's anything they need to respond to or address. Be concise and actionable.`

// BuildReviewPrompt constructs the prompt for PR summarization.
// When isMyPR is true, it generates author-focused guidance instead of reviewer guidance.
func BuildReviewPrompt(pr models.TrackedPR, files []models.ChangedFile) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# PR: %s\n", pr.Title))
	sb.WriteString(fmt.Sprintf("Repository: %s | Author: %s | %s -> %s\n", pr.Repo, pr.Author, pr.HeadBranch, pr.BaseBranch))
	sb.WriteString(fmt.Sprintf("Changed files: %d | +%d -%d\n\n", pr.ChangedFilesCount, pr.Additions, pr.Deletions))

	if pr.Body != "" {
		sb.WriteString("## PR Description\n")
		// Truncate very long descriptions
		body := pr.Body
		if len(body) > 2000 {
			body = body[:2000] + "\n... (truncated)"
		}
		sb.WriteString(body)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Changed Files\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("- %s (%s, +%d -%d)\n", f.Filename, f.Status, f.Additions, f.Deletions))
	}
	sb.WriteString("\n")

	// Include patches for smaller PRs (limit total diff size)
	totalPatchSize := 0
	const maxPatchSize = 30000
	var patchFiles []models.ChangedFile
	for _, f := range files {
		if f.Patch == "" {
			continue
		}
		totalPatchSize += len(f.Patch)
		if totalPatchSize > maxPatchSize {
			sb.WriteString("\n(Diff truncated — too large for full analysis)\n")
			break
		}
		patchFiles = append(patchFiles, f)
	}

	if len(patchFiles) > 0 {
		sb.WriteString("## Diff\n")
		for _, f := range patchFiles {
			sb.WriteString(fmt.Sprintf("### %s\n```diff\n%s\n```\n\n", f.Filename, f.Patch))
		}
	}

	if pr.IsMyPR {
		sb.WriteString(`This is YOUR PR. Please provide:
1. **Summary** (2-3 sentences): What does this PR do?
2. **Key Areas**: What parts of this PR might draw reviewer questions or concerns?
3. **Suggestions**: Anything you should proactively address or clarify before reviewers ask?

Keep each section brief. Focus on helping the author prepare for review feedback.`)
	} else {
		sb.WriteString(`Please provide:
1. **Summary** (2-3 sentences): What does this PR do?
2. **Key Review Areas**: What specific parts should a reviewer focus on? Reference file names and concepts.
3. **Potential Risks**: Any concerns about correctness, performance, security, or backward compatibility?

Keep each section brief and actionable.`)
	}

	return sb.String()
}
