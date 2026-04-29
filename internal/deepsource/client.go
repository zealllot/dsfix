package deepsource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
)

const (
	DefaultAPIEndpoint = "https://api.deepsource.io/graphql/"
)

// Client is a DeepSource API client
type Client struct {
	apiToken   string
	endpoint   string
	httpClient *http.Client
}

// NewClient creates a new DeepSource API client
func NewClient(apiToken string) *Client {
	return &Client{
		apiToken:   apiToken,
		endpoint:   DefaultAPIEndpoint,
		httpClient: &http.Client{},
	}
}

// graphqlRequest represents a GraphQL request
type graphqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphqlResponse represents a GraphQL response
type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

// doRequest executes a GraphQL request
func (c *Client) doRequest(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	reqBody := graphqlRequest{
		Query:     query,
		Variables: variables,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data, nil
}

// pathAllowed reports whether path passes the include/exclude glob filters.
// If include is non-empty, the path must match at least one include pattern.
// Then, the path is rejected if it matches any exclude pattern.
func pathAllowed(path string, include, exclude []string) bool {
	if len(include) > 0 {
		matched := false
		for _, pat := range include {
			if matchPath(pat, path) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	for _, pat := range exclude {
		if matchPath(pat, path) {
			return false
		}
	}
	return true
}

// matchPath is a slash-aware glob matcher that supports `**` (zero or more
// path segments) in addition to filepath.Match's per-segment globbing.
func matchPath(pattern, path string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(path, "/"))
}

func matchSegments(pat, path []string) bool {
	if len(pat) == 0 {
		return len(path) == 0
	}
	if pat[0] == "**" {
		for j := 0; j <= len(path); j++ {
			if matchSegments(pat[1:], path[j:]) {
				return true
			}
		}
		return false
	}
	if len(path) == 0 {
		return false
	}
	ok, _ := filepath.Match(pat[0], path[0])
	if !ok {
		return false
	}
	return matchSegments(pat[1:], path[1:])
}

// Occurrence represents a single occurrence
type Occurrence struct {
	ID        string
	Path      string
	BeginLine int
	EndLine   int
}

// fetchOccurrences fetches all occurrences for a repository issue with pagination
func (c *Client) fetchOccurrences(ctx context.Context, issueID string) ([]Occurrence, error) {
	var allOccurrences []Occurrence
	var cursor *string

	for {
		variables := map[string]any{
			"issueId": issueID,
			"first":   100,
		}
		if cursor != nil {
			variables["after"] = *cursor
		}

		data, err := c.doRequest(ctx, GetOccurrencesQuery, variables)
		if err != nil {
			return nil, err
		}

		var result struct {
			Node struct {
				Occurrences struct {
					Edges []struct {
						Node struct {
							ID        string `json:"id"`
							Path      string `json:"path"`
							BeginLine int    `json:"beginLine"`
							EndLine   int    `json:"endLine"`
						} `json:"node"`
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"occurrences"`
			} `json:"node"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to parse occurrences: %w", err)
		}

		for _, edge := range result.Node.Occurrences.Edges {
			allOccurrences = append(allOccurrences, Occurrence{
				ID:        edge.Node.ID,
				Path:      edge.Node.Path,
				BeginLine: edge.Node.BeginLine,
				EndLine:   edge.Node.EndLine,
			})
		}

		if !result.Node.Occurrences.PageInfo.HasNextPage {
			break
		}
		cursor = &result.Node.Occurrences.PageInfo.EndCursor
	}

	return allOccurrences, nil
}

// FetchIssues fetches issues from a repository
func (c *Client) FetchIssues(ctx context.Context, owner, repo string, filter *IssueFilter) ([]Issue, error) {
	var allIssues []Issue
	var cursor *string
	pageSize := 50 // Fixed page size for API requests
	maxIssues := 0
	if filter != nil && filter.Limit > 0 {
		maxIssues = filter.Limit
	}

	for {
		variables := map[string]any{
			"owner":    owner,
			"name":     repo,
			"provider": "GITHUB",
			"first":    pageSize,
		}
		if cursor != nil {
			variables["after"] = *cursor
		}

		data, err := c.doRequest(ctx, GetRepositoryIssuesQuery, variables)
		if err != nil {
			return nil, err
		}

		var result struct {
			Repository struct {
				Issues struct {
					Edges []struct {
						Node struct {
							ID    string `json:"id"`
							Issue struct {
								Shortcode   string `json:"shortcode"`
								Title       string `json:"title"`
								Category    string `json:"category"`
								Severity    string `json:"severity"`
								Description string `json:"description"`
								Analyzer    struct {
									Shortcode string `json:"shortcode"`
								} `json:"analyzer"`
							} `json:"issue"`
							Occurrences struct {
								Edges []struct {
									Node struct {
										ID        string `json:"id"`
										Path      string `json:"path"`
										BeginLine int    `json:"beginLine"`
										EndLine   int    `json:"endLine"`
									} `json:"node"`
								} `json:"edges"`
								PageInfo struct {
									HasNextPage bool   `json:"hasNextPage"`
									EndCursor   string `json:"endCursor"`
								} `json:"pageInfo"`
							} `json:"occurrences"`
						} `json:"node"`
					} `json:"edges"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"issues"`
			} `json:"repository"`
		}

		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("failed to parse issues: %w", err)
		}

		for _, edge := range result.Repository.Issues.Edges {
			node := edge.Node

			// Collect occurrences from first page
			var occurrences []Occurrence
			for _, occ := range node.Occurrences.Edges {
				occurrences = append(occurrences, Occurrence{
					ID:        occ.Node.ID,
					Path:      occ.Node.Path,
					BeginLine: occ.Node.BeginLine,
					EndLine:   occ.Node.EndLine,
				})
			}

			// If there are more occurrences, fetch them with pagination
			if node.Occurrences.PageInfo.HasNextPage {
				moreOccurrences, err := c.fetchOccurrences(ctx, node.ID)
				if err != nil {
					return nil, fmt.Errorf("failed to fetch additional occurrences: %w", err)
				}
				// Replace with full list (fetchOccurrences gets all pages)
				occurrences = moreOccurrences
			}

			for _, occ := range occurrences {
				issue := Issue{
					ID:          occ.ID,
					Title:       node.Issue.Title,
					Category:    node.Issue.Category,
					Shortcode:   node.Issue.Shortcode,
					Severity:    node.Issue.Severity,
					FilePath:    occ.Path,
					BeginLine:   occ.BeginLine,
					EndLine:     occ.EndLine,
					Description: node.Issue.Description,
					Analyzer:    node.Issue.Analyzer.Shortcode,
				}

				if filter != nil {
					if len(filter.Categories) > 0 && !slices.Contains(filter.Categories, issue.Category) {
						continue
					}
					if len(filter.Severities) > 0 && !slices.Contains(filter.Severities, issue.Severity) {
						continue
					}
					if !pathAllowed(issue.FilePath, filter.PathsInclude, filter.PathsExclude) {
						continue
					}
				}

				allIssues = append(allIssues, issue)

				// Check limit after each issue added
				if maxIssues > 0 && len(allIssues) >= maxIssues {
					return allIssues[:maxIssues], nil
				}
			}
		}

		if !result.Repository.Issues.PageInfo.HasNextPage {
			break
		}
		cursor = &result.Repository.Issues.PageInfo.EndCursor

		// Respect limit if set
		if maxIssues > 0 && len(allIssues) >= maxIssues {
			allIssues = allIssues[:maxIssues]
			break
		}
	}

	return allIssues, nil
}
