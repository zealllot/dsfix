package cascade

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zealllot/dsfix/internal/task"
)

// Executor handles the execution of fix tasks
type Executor struct {
	manager  *task.Manager
	repoPath string
}

// NewExecutor creates a new executor
func NewExecutor(manager *task.Manager, repoPath string) *Executor {
	return &Executor{
		manager:  manager,
		repoPath: repoPath,
	}
}

// RunInteractive runs the fix process interactively
func (e *Executor) RunInteractive() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Get next pending task
		t := e.manager.GetNextTask()
		if t == nil {
			fmt.Println("\n✅ All tasks completed!")
			e.printStats()
			return nil
		}

		// Mark as in progress
		t, err := e.manager.StartTask(t.ID)
		if err != nil {
			return fmt.Errorf("failed to start task: %w", err)
		}

		// Display task info
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println(GenerateFixPrompt(t, e.repoPath))
		fmt.Println(strings.Repeat("=", 60))

		// Wait for user action
		fmt.Println("\nActions:")
		fmt.Println("  [f] Mark as fixed (will prompt for commit)")
		fmt.Println("  [s] Skip this task")
		fmt.Println("  [q] Quit")
		fmt.Print("\nEnter action: ")

		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		action := strings.TrimSpace(strings.ToLower(input))

		switch action {
		case "f", "fixed":
			if err := e.handleFixed(t, reader); err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
		case "s", "skip":
			fmt.Print("Reason for skipping: ")
			reason, _ := reader.ReadString('\n')
			if err := e.manager.SkipTask(t.ID, strings.TrimSpace(reason)); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			fmt.Println("Task skipped.")
		case "q", "quit":
			fmt.Println("Exiting...")
			e.printStats()
			return nil
		default:
			fmt.Println("Invalid action. Please try again.")
			// Reset task to pending
			t.Status = task.StatusPending
		}
	}
}

// handleFixed handles the fixed action
func (e *Executor) handleFixed(t *task.Task, reader *bufio.Reader) error {
	// Suggest commit message
	suggestedMsg := t.GenerateCommitMessage()
	fmt.Printf("\nSuggested commit message: %s\n", suggestedMsg)
	fmt.Print("Press Enter to use suggested message, or type a custom message: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	commitMsg := strings.TrimSpace(input)
	if commitMsg == "" {
		commitMsg = suggestedMsg
	}

	// Stage the file
	stageCmd := exec.Command("git", "add", t.Issue.FilePath)
	stageCmd.Dir = e.repoPath
	if err := stageCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	// Create commit
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = e.repoPath
	output, err := commitCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit: %w\n%s", err, string(output))
	}

	// Get commit hash
	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	hashCmd.Dir = e.repoPath
	hashOutput, err := hashCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commit hash: %w", err)
	}
	commitHash := strings.TrimSpace(string(hashOutput))

	// Mark task as fixed
	if err := e.manager.CompleteTask(t.ID, commitHash, commitMsg); err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	fmt.Printf("\n✅ Task fixed! Commit: %s\n", commitHash[:8])
	return nil
}

// printStats prints the current statistics
func (e *Executor) printStats() {
	fmt.Println("\n" + GenerateProgressReport(e.manager.GetStats()))
}

// OutputTaskForCascade outputs the current task in a format suitable for Cascade
func (e *Executor) OutputTaskForCascade() (*task.Task, error) {
	t := e.manager.GetNextTask()
	if t == nil {
		return nil, nil
	}

	// Mark as in progress
	t, err := e.manager.StartTask(t.ID)
	if err != nil {
		return nil, err
	}

	// Output the prompt
	fmt.Println(GenerateFixPrompt(t, e.repoPath))

	return t, nil
}
