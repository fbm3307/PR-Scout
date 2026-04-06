package github

import (
	"strings"

	"github.com/codeready-toolchain/pr-scout/pkg/models"
)

const codeRabbitBotLogin = "coderabbitai[bot]"

// FilterCodeRabbitComments extracts CodeRabbit bot comments from a list of review comments.
func FilterCodeRabbitComments(comments []models.ReviewComment) []models.ReviewComment {
	var crComments []models.ReviewComment
	for _, c := range comments {
		if isCodeRabbitComment(c) {
			crComments = append(crComments, c)
		}
	}
	return crComments
}

// SummarizeCodeRabbitFindings produces a brief summary of CodeRabbit's findings.
func SummarizeCodeRabbitFindings(comments []models.ReviewComment) string {
	crComments := FilterCodeRabbitComments(comments)
	if len(crComments) == 0 {
		return ""
	}

	var fileComments []models.ReviewComment
	hasActionableContent := false
	isCleanReview := false

	for _, c := range crComments {
		if c.FilePath != "" {
			fileComments = append(fileComments, c)
			continue
		}
		lower := strings.ToLower(c.Body)
		if strings.Contains(lower, "no actionable comments") {
			isCleanReview = true
		} else if strings.Contains(lower, "actionable comments") ||
			strings.Contains(lower, "nitpick") {
			hasActionableContent = true
		}
	}

	if len(fileComments) > 0 {
		seen := make(map[string]bool)
		var files []string
		for _, c := range fileComments {
			if !seen[c.FilePath] {
				files = append(files, c.FilePath)
				seen[c.FilePath] = true
			}
		}

		var sb strings.Builder
		sb.WriteString("CodeRabbit found ")
		if len(fileComments) == 1 {
			sb.WriteString("1 issue")
		} else {
			sb.WriteString(strings.Join([]string{itoa(len(fileComments)), " issues"}, ""))
		}
		sb.WriteString(" across ")
		if len(files) == 1 {
			sb.WriteString("1 file")
		} else {
			sb.WriteString(strings.Join([]string{itoa(len(files)), " files"}, ""))
		}
		sb.WriteString(".")
		return sb.String()
	}

	if hasActionableContent {
		return "CodeRabbit found issues."
	}

	if isCleanReview {
		return "CodeRabbit reviewed — no issues found."
	}

	return "CodeRabbit reviewed."
}

func isCodeRabbitComment(c models.ReviewComment) bool {
	return c.IsBot && strings.EqualFold(c.Commenter, codeRabbitBotLogin)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
