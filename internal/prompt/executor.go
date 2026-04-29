package prompt

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zealllot/dsfix/internal/task"
)

// Executor handles the interactive fix process.
type Executor struct {
	manager   *task.Manager
	repoPath  string
	verifyCmd string
}

// NewExecutor creates a new executor.
func NewExecutor(manager *task.Manager, repoPath, verifyCmd string) *Executor {
	return &Executor{
		manager:   manager,
		repoPath:  repoPath,
		verifyCmd: verifyCmd,
	}
}

// RunInteractive runs the legacy interactive fix loop (one task at a time, terminal-driven).
func (e *Executor) RunInteractive() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		t := e.manager.GetNextTask()
		if t == nil {
			fmt.Println("\n✅ All tasks completed!")
			e.printStats()
			return nil
		}

		t, err := e.manager.StartTask(t.ID)
		if err != nil {
			return fmt.Errorf("failed to start task: %w", err)
		}

		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println(GenerateFixPrompt(t, e.verifyCmd))
		fmt.Println(strings.Repeat("=", 60))

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
			if err := e.manager.RevertToPending(t.ID); err != nil {
				fmt.Printf("Warning: failed to revert task status: %v\n", err)
			}
		}
	}
}

// handleFixed handles the fixed action: stage, commit, mark complete.
func (e *Executor) handleFixed(t *task.Task, reader *bufio.Reader) error {
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

	stageCmd := exec.Command("git", "add", t.Issue.FilePath)
	stageCmd.Dir = e.repoPath
	if err := stageCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = e.repoPath
	output, err := commitCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit: %w\n%s", err, string(output))
	}

	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	hashCmd.Dir = e.repoPath
	hashOutput, err := hashCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commit hash: %w", err)
	}
	commitHash := strings.TrimSpace(string(hashOutput))

	if err := e.manager.CompleteTask(t.ID, commitHash, commitMsg); err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	fmt.Printf("\n✅ Task fixed! Commit: %s\n", commitHash[:8])
	return nil
}

func (e *Executor) printStats() {
	fmt.Println("\n" + GenerateProgressReport(e.manager.GetStats()))
}

// OutputTask prints the next pending task's prompt and marks it in_progress.
func (e *Executor) OutputTask() (*task.Task, error) {
	t := e.manager.GetNextTask()
	if t == nil {
		return nil, nil
	}

	t, err := e.manager.StartTask(t.ID)
	if err != nil {
		return nil, err
	}

	fmt.Println(GenerateFixPrompt(t, e.verifyCmd))

	return t, nil
}
