package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const graphqlEndpoint = "https://api.github.com/graphql"

// CodeRabbitResolution tracks how many CodeRabbit review threads are resolved.
type CodeRabbitResolution struct {
	Total    int
	Resolved int
}

// FetchCodeRabbitResolution queries the GitHub GraphQL API for review thread
// resolution status and returns counts for threads started by coderabbitai[bot].
// Uses raw HTTP with the existing OAuth token -- no third-party GraphQL library needed.
func (c *Client) FetchCodeRabbitResolution(ctx context.Context, repo string, prNumber int) (*CodeRabbitResolution, error) {
	query := `query($owner: String!, $repo: String!, $number: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $number) {
				reviewThreads(first: 100) {
					nodes {
						isResolved
						comments(first: 1) {
							nodes {
								author { login }
							}
						}
					}
				}
			}
		}
	}`

	body := graphqlRequest{
		Query: query,
		Variables: map[string]any{
			"owner":  c.org,
			"repo":   repo,
			"number": prNumber,
		},
	}

	respData, err := c.doGraphQL(ctx, body)
	if err != nil {
		return nil, err
	}

	result := &CodeRabbitResolution{}
	threads := respData.Data.Repository.PullRequest.ReviewThreads.Nodes
	for _, t := range threads {
		if len(t.Comments.Nodes) == 0 {
			continue
		}
		author := t.Comments.Nodes[0].Author.Login
		if !strings.EqualFold(author, "coderabbitai[bot]") && !strings.EqualFold(author, "coderabbitai") {
			continue
		}
		result.Total++
		if t.IsResolved {
			result.Resolved++
		}
	}

	return result, nil
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse struct {
	Data   graphqlData    `json:"data"`
	Errors []graphqlError `json:"errors,omitempty"`
}

type graphqlError struct {
	Message string `json:"message"`
}

type graphqlData struct {
	Repository struct {
		PullRequest struct {
			ReviewThreads struct {
				Nodes []reviewThreadNode `json:"nodes"`
			} `json:"reviewThreads"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

type reviewThreadNode struct {
	IsResolved bool `json:"isResolved"`
	Comments   struct {
		Nodes []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
		} `json:"nodes"`
	} `json:"comments"`
}

func (c *Client) doGraphQL(ctx context.Context, reqBody graphqlRequest) (*graphqlResponse, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal graphql request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlEndpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read graphql response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var result graphqlResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("decode graphql response: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("graphql errors: %s", result.Errors[0].Message)
	}

	return &result, nil
}
