package task

import (
"context"
"fmt"
"sort"

"github.com/zealllot/dsfix/internal/deepsource"
)

// Manager handles task operations
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

// GetNextTask returns the next task to process (pending or in_progress)
func (m *Manager) GetNextTask() *Task {
	// First check in_progress tasks
	inProgress := m.store.GetByStatus(StatusInProgress)
	if len(inProgress) > 0 {
		return inProgress[0]
	}
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

// GetPendingTasks returns all pending and in_progress tasks (tasks that need to be processed)
func (m *Manager) GetPendingTasks() []*Task {
	pending := m.store.GetByStatus(StatusPending)
	inProgress := m.store.GetByStatus(StatusInProgress)
	return append(pending, inProgress...)
}

// GetTasksByShortcode returns all pending and in_progress tasks with the specified shortcode
func (m *Manager) GetTasksByShortcode(shortcode string) []*Task {
	pending := m.store.GetByStatus(StatusPending)
	inProgress := m.store.GetByStatus(StatusInProgress)
	allTasks := append(pending, inProgress...)
	
	var filtered []*Task
	for _, t := range allTasks {
		if t.Issue.Shortcode == shortcode {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// GetShortcodeStats returns statistics grouped by shortcode for pending and in_progress tasks
func (m *Manager) GetShortcodeStats() map[string]int {
	pending := m.store.GetByStatus(StatusPending)
	inProgress := m.store.GetByStatus(StatusInProgress)
	tasks := append(pending, inProgress...)
	
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

// ResetInProgress resets all in-progress tasks back to pending
func (m *Manager) ResetInProgress() (int, error) {
	tasks := m.store.GetByStatus(StatusInProgress)
	for _, task := range tasks {
		task.Status = StatusPending
		m.store.Update(task)
	}
	if err := m.store.Save(); err != nil {
		return 0, err
	}
	return len(tasks), nil
}
