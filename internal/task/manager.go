package task

import (
"context"
"fmt"
"sort"

"github.com/zealllot/dsfix/internal/deepsource"
)

// Manager manages the task lifecycle
type Manager struct {
	store  *Store
	client *deepsource.Client
	owner  string
	repo   string
}

// NewManager creates a new task manager
func NewManager(store *Store, client *deepsource.Client, owner, repo string) *Manager {
	return &Manager{
		store:  store,
		client: client,
		owner:  owner,
		repo:   repo,
	}
}

// Sync fetches issues from DeepSource and creates tasks
func (m *Manager) Sync(ctx context.Context, filter *deepsource.IssueFilter) (int, error) {
	issues, err := m.client.FetchIssues(ctx, m.owner, m.repo, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch issues: %w", err)
	}

	newCount := 0
	for _, issue := range issues {
		// Skip if task already exists
		if _, exists := m.store.Get(issue.ID); exists {
			continue
		}

		task := NewTask(issue)
		m.store.Add(task)
		newCount++
	}

	if err := m.store.Save(); err != nil {
		return newCount, fmt.Errorf("failed to save tasks: %w", err)
	}

	return newCount, nil
}

// GetNextTask returns the next task to process
func (m *Manager) GetNextTask() *Task {
	return m.store.GetNextPending()
}

// StartTask marks a task as in progress
func (m *Manager) StartTask(taskID string) (*Task, error) {
	task, ok := m.store.Get(taskID)
	if !ok {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	task.MarkInProgress()
	m.store.Update(task)

	if err := m.store.Save(); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	return task, nil
}

// CompleteTask marks a task as fixed
func (m *Manager) CompleteTask(taskID, commitHash, commitMsg string) error {
	task, ok := m.store.Get(taskID)
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.MarkFixed(commitHash, commitMsg)
	m.store.Update(task)

	return m.store.Save()
}

// SkipTask marks a task as skipped
func (m *Manager) SkipTask(taskID, reason string) error {
	task, ok := m.store.Get(taskID)
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.MarkSkipped(reason)
	m.store.Update(task)

	return m.store.Save()
}

// FailTask marks a task as failed
func (m *Manager) FailTask(taskID, errMsg string) error {
	task, ok := m.store.Get(taskID)
	if !ok {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.MarkFailed(errMsg)
	m.store.Update(task)

	return m.store.Save()
}

// GetStats returns task statistics
func (m *Manager) GetStats() map[Status]int {
	return m.store.Count()
}

// GetAllTasks returns all tasks sorted by priority
func (m *Manager) GetAllTasks() []*Task {
	tasks := m.store.GetAll()

	// Sort by severity (critical > major > minor) then by category
	severityOrder := map[string]int{
		"critical": 0,
		"major":    1,
		"minor":    2,
	}

	sort.Slice(tasks, func(i, j int) bool {
si := severityOrder[tasks[i].Issue.Severity]
sj := severityOrder[tasks[j].Issue.Severity]
if si != sj {
return si < sj
		}
		return tasks[i].Issue.Category < tasks[j].Issue.Category
	})

	return tasks
}

// GetPendingTasks returns all pending tasks
func (m *Manager) GetPendingTasks() []*Task {
	return m.store.GetByStatus(StatusPending)
}

// GetTasksByShortcode returns all pending tasks with the specified shortcode
func (m *Manager) GetTasksByShortcode(shortcode string) []*Task {
	allTasks := m.store.GetByStatus(StatusPending)
	var filtered []*Task
	for _, t := range allTasks {
		if t.Issue.Shortcode == shortcode {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// GetShortcodeStats returns statistics grouped by shortcode for pending tasks
func (m *Manager) GetShortcodeStats() map[string]int {
	tasks := m.store.GetByStatus(StatusPending)
	stats := make(map[string]int)
	for _, t := range tasks {
		stats[t.Issue.Shortcode]++
	}
	return stats
}

// StartBatch marks multiple tasks as in progress
func (m *Manager) StartBatch(taskIDs []string) error {
	for _, id := range taskIDs {
		task, ok := m.store.Get(id)
		if !ok {
			continue
		}
		task.MarkInProgress()
		m.store.Update(task)
	}
	return m.store.Save()
}

// CompleteBatch marks multiple tasks as fixed
func (m *Manager) CompleteBatch(taskIDs []string, commitHash, commitMsg string) error {
	for _, id := range taskIDs {
		task, ok := m.store.Get(id)
		if !ok {
			continue
		}
		task.MarkFixed(commitHash, commitMsg)
		m.store.Update(task)
	}
	return m.store.Save()
}

// SkipBatch marks multiple tasks as skipped
func (m *Manager) SkipBatch(taskIDs []string, reason string) error {
	for _, id := range taskIDs {
		task, ok := m.store.Get(id)
		if !ok {
			continue
		}
		task.MarkSkipped(reason)
		m.store.Update(task)
	}
	return m.store.Save()
}

// GetInProgressTasks returns all in-progress tasks
func (m *Manager) GetInProgressTasks() []*Task {
	return m.store.GetByStatus(StatusInProgress)
}

// Reset resets all tasks to pending status
func (m *Manager) Reset() error {
	tasks := m.store.GetAll()
	for _, task := range tasks {
		task.Status = StatusPending
		task.CommitHash = ""
		task.CommitMsg = ""
		task.FixedAt = nil
		task.ErrorMsg = ""
		m.store.Update(task)
	}
	return m.store.Save()
}
