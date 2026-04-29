package deepsource

import "time"

// Issue represents a DeepSource issue
type Issue struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	Shortcode   string    `json:"shortcode"`
	Severity    string    `json:"severity"`
	FilePath    string    `json:"file_path"`
	BeginLine   int       `json:"begin_line"`
	EndLine     int       `json:"end_line"`
	Description string    `json:"description"`
	Suggestion  string    `json:"suggestion"`
	Analyzer    string    `json:"analyzer"`
	CreatedAt   time.Time `json:"created_at"`
}

// IssueFilter defines filters for fetching issues
type IssueFilter struct {
	Categories   []string // Bug Risk, Anti-pattern, Security, Performance, etc.
	Severities   []string // critical, major, minor
	Limit        int
	Offset       int
	PathsInclude []string // glob patterns; if non-empty, occurrence path must match one
	PathsExclude []string // glob patterns; occurrence is dropped if path matches one
}

// Repository represents a DeepSource repository
type Repository struct {
	Name          string `json:"name"`
	DefaultBranch string `json:"default_branch"`
	Provider      string `json:"provider"` // github, gitlab, bitbucket
}

// AnalysisRun represents a DeepSource analysis run
type AnalysisRun struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	CommitSHA string    `json:"commit_sha"`
}
