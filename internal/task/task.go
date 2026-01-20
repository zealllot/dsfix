package task

import (
	"time"

	"github.com/zealllot/dsfix/internal/deepsource"
)

// Status represents the status of a task
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusFixed      Status = "fixed"
	StatusSkipped    Status = "skipped"
	StatusFailed     Status = "failed"
)

// Task represents a fix task derived from a DeepSource issue
type Task struct {
	ID         string           `json:"id"`
	Issue      deepsource.Issue `json:"issue"`
	Status     Status           `json:"status"`
	CommitHash string           `json:"commit_hash,omitempty"`
	CommitMsg  string           `json:"commit_msg,omitempty"`
	FixedAt    *time.Time       `json:"fixed_at,omitempty"`
	ErrorMsg   string           `json:"error_msg,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// NewTask creates a new task from a DeepSource issue
func NewTask(issue deepsource.Issue) *Task {
	now := time.Now()
	return &Task{
		ID:        issue.ID,
		Issue:     issue,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// MarkInProgress marks the task as in progress
func (t *Task) MarkInProgress() {
	t.Status = StatusInProgress
	t.UpdatedAt = time.Now()
}

// MarkFixed marks the task as fixed with commit info
func (t *Task) MarkFixed(commitHash, commitMsg string) {
	now := time.Now()
	t.Status = StatusFixed
	t.CommitHash = commitHash
	t.CommitMsg = commitMsg
	t.FixedAt = &now
	t.UpdatedAt = now
}

// MarkSkipped marks the task as skipped
func (t *Task) MarkSkipped(reason string) {
	t.Status = StatusSkipped
	t.ErrorMsg = reason
	t.UpdatedAt = time.Now()
}

// MarkFailed marks the task as failed
func (t *Task) MarkFailed(err string) {
	t.Status = StatusFailed
	t.ErrorMsg = err
	t.UpdatedAt = time.Now()
}

// GenerateCommitMessage generates a suggested commit message for this task
func (t *Task) GenerateCommitMessage() string {
	return "fix(" + t.Issue.Shortcode + "): " + t.Issue.Title
}
