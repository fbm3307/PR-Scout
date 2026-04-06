// Package llm provides an LLM client for PR summarization via Anthropic Vertex AI.
package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/vertex"

	"github.com/codeready-toolchain/pr-scout/pkg/config"
	"github.com/codeready-toolchain/pr-scout/pkg/models"
)

// Client wraps the Anthropic Vertex AI client for PR summarization.
type Client struct {
	client *anthropic.Client
	model  string
	logger *slog.Logger
}

// ReviewAnalysis holds the structured output from LLM analysis.
type ReviewAnalysis struct {
	Summary     string `json:"summary"`
	ReviewHints string `json:"review_hints"`
	RiskNotes   string `json:"risk_notes"`
}

// NewClient creates an LLM client configured for Anthropic via Vertex AI.
// Returns nil if LLM is not configured (optional feature).
func NewClient(cfg config.LLMConfig, logger *slog.Logger) *Client {
	if !cfg.Enabled {
		logger.Info("LLM disabled — PR summaries will not be generated")
		return nil
	}

	projectID := cfg.ProjectID()
	if projectID == "" {
		logger.Warn("LLM enabled but ANTHROPIC_VERTEX_PROJECT_ID not set — disabling")
		return nil
	}

	client := anthropic.NewClient(
		vertex.WithGoogleAuth(context.Background(), cfg.Region, projectID),
	)

	logger.Info("LLM client initialized",
		"provider", cfg.Provider, "model", cfg.Model, "region", cfg.Region)

	return &Client{
		client: &client,
		model:  cfg.Model,
		logger: logger,
	}
}

// AnalyzePR generates a review summary for the given PR and its changed files.
func (c *Client) AnalyzePR(ctx context.Context, pr models.TrackedPR, files []models.ChangedFile) (*ReviewAnalysis, error) {
	prompt := BuildReviewPrompt(pr, files)

	sysPrompt := reviewerSystemPrompt
	if pr.IsMyPR {
		sysPrompt = authorSystemPrompt
	}

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: sysPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("llm request failed: %w", err)
	}

	// Extract text from response
	var responseText string
	for _, block := range resp.Content {
		if block.Type == "text" {
			responseText += block.Text
		}
	}

	return parseReviewAnalysis(responseText), nil
}

// parseReviewAnalysis splits the LLM response into structured sections.
func parseReviewAnalysis(text string) *ReviewAnalysis {
	analysis := &ReviewAnalysis{}

	sections := map[string]*string{
		"summary":          &analysis.Summary,
		"key review areas": &analysis.ReviewHints,
		"potential risks":  &analysis.RiskNotes,
	}

	lines := strings.Split(text, "\n")
	var currentField *string

	for _, line := range lines {
		lower := strings.ToLower(line)

		// Check if this line starts a new section
		matched := false
		for keyword, field := range sections {
			if strings.Contains(lower, keyword) && strings.Contains(line, "**") {
				currentField = field
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		if currentField != nil {
			if *currentField != "" {
				*currentField += "\n"
			}
			*currentField += line
		}
	}

	// Trim all fields
	analysis.Summary = strings.TrimSpace(analysis.Summary)
	analysis.ReviewHints = strings.TrimSpace(analysis.ReviewHints)
	analysis.RiskNotes = strings.TrimSpace(analysis.RiskNotes)

	// Fallback: if parsing failed, put everything in summary
	if analysis.Summary == "" && analysis.ReviewHints == "" {
		analysis.Summary = strings.TrimSpace(text)
	}

	return analysis
}
