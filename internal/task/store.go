package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const DefaultStoreFile = ".dsfix/tasks.json"

// Store manages task persistence
type Store struct {
	mu       sync.RWMutex
	filePath string
	tasks    map[string]*Task
}

// StoreData represents the JSON structure for persistence
type StoreData struct {
	Tasks   []*Task `json:"tasks"`
	Version string  `json:"version"`
}

// NewStore creates a new task store
func NewStore(baseDir string) (*Store, error) {
	filePath := filepath.Join(baseDir, DefaultStoreFile)

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	store := &Store{
		filePath: filePath,
		tasks:    make(map[string]*Task),
	}

	// Load existing tasks if file exists
	if _, err := os.Stat(filePath); err == nil {
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("failed to load existing tasks: %w", err)
		}
	}

	return store, nil
}

// load reads tasks from the store file
func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var storeData StoreData
	if err := json.Unmarshal(data, &storeData); err != nil {
		return err
	}

	s.tasks = make(map[string]*Task)
	for _, task := range storeData.Tasks {
		s.tasks[task.ID] = task
	}

	return nil
}

// Save persists tasks to the store file
func (s *Store) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}

	storeData := StoreData{
		Tasks:   tasks,
		Version: "1.0",
	}

	data, err := json.MarshalIndent(storeData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write store file: %w", err)
	}

	return nil
}

// Add adds a new task to the store
func (s *Store) Add(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
}

// Get retrieves a task by ID
func (s *Store) Get(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	return task, ok
}

// Update updates an existing task
func (s *Store) Update(task *Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
}

// GetAll returns all tasks
func (s *Store) GetAll() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

// GetByStatus returns tasks with the specified status
func (s *Store) GetByStatus(status Status) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*Task
	for _, task := range s.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetNextPending returns the next pending task
func (s *Store) GetNextPending() *Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, task := range s.tasks {
		if task.Status == StatusPending {
			return task
		}
	}
	return nil
}

// Count returns the count of tasks by status
func (s *Store) Count() map[Status]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[Status]int)
	for _, task := range s.tasks {
		counts[task.Status]++
	}
	return counts
}

// Clear removes all tasks
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = make(map[string]*Task)
}
