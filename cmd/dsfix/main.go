package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zealllot/dsfix/config"
	"github.com/zealllot/dsfix/internal/claudeinit"
	"github.com/zealllot/dsfix/internal/deepsource"
	"github.com/zealllot/dsfix/internal/prompt"
	"github.com/zealllot/dsfix/internal/task"
)

var (
	cfgFile  string
	repoPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dsfix",
		Short: "DeepSource fix workflow tool",
		Long:  "DSFix turns DeepSource issues into per-shortcode batch tasks for an AI assistant to fix.",
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .dsfix.yaml)")
	rootCmd.PersistentFlags().StringVarP(&repoPath, "repo", "r", "", "repository path (default is current directory)")

	rootCmd.AddCommand(
		initCmd(),
		initClaudeCmd(),
		syncCmd(),
		runCmd(),
		statusCmd(),
		nextCmd(),
		completeCmd(),
		skipCmd(),
		resetCmd(),
		resetProgressCmd(),
		listCmd(),
		batchCmd(),
		completeBatchCmd(),
		skipBatchCmd(),
		startCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// session bundles the per-command state every command needs.
type session struct {
	cfg     *config.Config
	manager *task.Manager
	repo    string
}

// newSession loads config, opens the task store, and builds a manager.
// It does NOT call Validate() — commands that need a valid token (sync, etc.) call it explicitly.
func newSession() (*session, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	repo := getRepoPath()
	store, err := task.NewStore(repo)
	if err != nil {
		return nil, err
	}
	client := deepsource.NewClient(cfg.DeepSource.APIToken)
	manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)
	return &session{cfg: cfg, manager: manager, repo: repo}, nil
}

func loadConfig() (*config.Config, error) {
	path := cfgFile
	if path == "" {
		base := repoPath
		if base == "" {
			base, _ = os.Getwd()
		}
		path = filepath.Join(base, config.DefaultConfigFile)
	}
	return config.Load(path)
}

func getRepoPath() string {
	if repoPath != "" {
		return repoPath
	}
	cwd, _ := os.Getwd()
	return cwd
}

func filterFromConfig(cfg *config.Config) *deepsource.IssueFilter {
	return &deepsource.IssueFilter{
		Categories:   cfg.Filter.Categories,
		Severities:   cfg.Filter.Severities,
		Limit:        cfg.Filter.Limit,
		PathsInclude: cfg.Filter.PathsInclude,
		PathsExclude: cfg.Filter.PathsExclude,
	}
}

func initClaudeCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init-claude",
		Short: "Scaffold Claude Code integration files (slash command + subagent)",
		RunE: func(cmd *cobra.Command, args []string) error {
			written, skipped, err := claudeinit.Scaffold(getRepoPath(), force)
			if err != nil {
				return err
			}
			for _, p := range written {
				fmt.Printf("✅ Wrote %s\n", p)
			}
			for _, p := range skipped {
				fmt.Printf("⏭  Skipped %s (already exists, pass --force to overwrite)\n", p)
			}
			if len(written) > 0 {
				fmt.Println("\nIn Claude Code, type /dsfix to start the workflow.")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files")
	return cmd
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize dsfix configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := filepath.Join(getRepoPath(), config.DefaultConfigFile)
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("config file already exists: %s", path)
			}
			if err := os.WriteFile(path, []byte(config.GenerateTemplate()), 0644); err != nil {
				return fmt.Errorf("failed to create config file: %w", err)
			}
			fmt.Printf("Created config file: %s\n", path)
			fmt.Println("Edit it and add your DeepSource API token (or set DEEPSOURCE_API_TOKEN env var).")
			return nil
		},
	}
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync issues from DeepSource",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}
			if err := s.cfg.Validate(); err != nil {
				return err
			}
			fmt.Println("Fetching issues from DeepSource...")
			count, err := s.manager.Sync(context.Background(), filterFromConfig(s.cfg))
			if err != nil {
				return err
			}
			fmt.Printf("Synced %d new issues.\n", count)
			printStats(s.manager)
			return nil
		},
	}
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run interactive fix process (legacy terminal-driven mode)",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}
			if err := s.cfg.Validate(); err != nil {
				return err
			}
			executor := prompt.NewExecutor(s.manager, s.repo, s.cfg.Verify.Command)
			return executor.RunInteractive()
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current status",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}
			printStats(s.manager)
			return nil
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List pending issues grouped by type",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}
			fmt.Println(prompt.GenerateGroupStats(s.manager.GetPendingTasks()))
			return nil
		},
	}
}

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Show task list and let the AI guide the fix flow",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}

			tasks := s.manager.GetPendingTasks()
			if len(tasks) == 0 {
				fmt.Println("No pending tasks. Syncing from DeepSource...")
				if err := s.cfg.Validate(); err != nil {
					return err
				}
				count, err := s.manager.Sync(context.Background(), filterFromConfig(s.cfg))
				if err != nil {
					return err
				}
				fmt.Printf("Synced %d new issues.\n\n", count)
			}

			fmt.Println(prompt.GenerateStartPrompt(s.manager.GetAllTasks()))
			return nil
		},
	}
}

func nextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Output the next task for the AI to fix",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}
			executor := prompt.NewExecutor(s.manager, s.repo, s.cfg.Verify.Command)
			t, err := executor.OutputTask()
			if err != nil {
				return err
			}
			if t == nil {
				fmt.Println("No pending tasks.")
			}
			return nil
		},
	}
}

func batchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "batch [shortcode]",
		Short: "Output all tasks of a specific type for batch fixing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shortcode := args[0]
			s, err := newSession()
			if err != nil {
				return err
			}

			tasks := s.manager.GetTasksByShortcode(shortcode)
			if len(tasks) == 0 {
				fmt.Printf("No pending tasks with shortcode: %s\n", shortcode)
				return nil
			}
			if limit > 0 && len(tasks) > limit {
				tasks = tasks[:limit]
			}

			taskIDs := make([]string, 0, len(tasks))
			for _, t := range tasks {
				taskIDs = append(taskIDs, t.ID)
			}
			if err := s.manager.StartBatch(taskIDs); err != nil {
				return err
			}

			fmt.Println(prompt.GenerateBatchFixPrompt(tasks, s.cfg.Verify.Command))
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "limit number of tasks (default: all)")
	return cmd
}

func completeCmd() *cobra.Command {
	var taskID string
	var commitMsg string
	var noCommit bool

	cmd := &cobra.Command{
		Use:   "complete",
		Short: "Mark current task as completed and commit changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}

			t, err := resolveInProgressTask(s.manager, taskID)
			if err != nil {
				return err
			}
			if commitMsg == "" {
				commitMsg = t.GenerateCommitMessage()
			}

			commitHash := "no-commit"
			if !noCommit {
				if err := stageFiles(s.repo, []string{t.Issue.FilePath}); err != nil {
					return err
				}
				h, err := gitCommit(s.repo, commitMsg)
				if err != nil {
					return err
				}
				commitHash = h
				fmt.Printf("✅ Committed: %s\n", commitHash)
			}

			if err := s.manager.CompleteTask(t.ID, commitHash, commitMsg); err != nil {
				return err
			}
			fmt.Printf("Task completed: %s\n", t.Issue.Title)
			fmt.Printf("Commit message: %s\n", commitMsg)
			return nil
		},
	}
	cmd.Flags().StringVarP(&taskID, "id", "i", "", "task ID (default: current in-progress task)")
	cmd.Flags().StringVarP(&commitMsg, "message", "m", "", "commit message")
	cmd.Flags().BoolVar(&noCommit, "no-commit", false, "skip git commit")
	return cmd
}

func completeBatchCmd() *cobra.Command {
	var commitMsg string
	var noCommit bool

	cmd := &cobra.Command{
		Use:   "complete-batch",
		Short: "Mark all in-progress tasks as completed and commit changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}

			tasks := s.manager.GetInProgressTasks()
			if len(tasks) == 0 {
				return fmt.Errorf("no tasks in progress")
			}

			files, taskIDs := uniqueFilesAndIDs(tasks)
			if commitMsg == "" {
				first := tasks[0]
				commitMsg = fmt.Sprintf("fix(%s): %s (%d occurrences)", first.Issue.Shortcode, first.Issue.Title, len(tasks))
			}

			commitHash := "no-commit"
			if !noCommit {
				if err := stageFiles(s.repo, files); err != nil {
					return err
				}
				h, err := gitCommit(s.repo, commitMsg)
				if err != nil {
					return err
				}
				commitHash = h
				fmt.Printf("✅ Committed: %s\n", commitHash)
			}

			if err := s.manager.CompleteBatch(taskIDs, commitHash, commitMsg); err != nil {
				return err
			}
			fmt.Printf("Batch completed: %d tasks\n", len(tasks))
			fmt.Printf("Files modified: %d\n", len(files))
			fmt.Printf("Commit message: %s\n", commitMsg)
			return nil
		},
	}
	cmd.Flags().StringVarP(&commitMsg, "message", "m", "", "commit message")
	cmd.Flags().BoolVar(&noCommit, "no-commit", false, "skip git commit")
	return cmd
}

func skipCmd() *cobra.Command {
	var taskID string
	var reason string
	var noRevert bool

	cmd := &cobra.Command{
		Use:   "skip",
		Short: "Skip current task and revert changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}

			t, err := resolveInProgressTask(s.manager, taskID)
			if err != nil {
				return err
			}
			if !noRevert {
				if err := gitRevert(s.repo, []string{t.Issue.FilePath}); err != nil {
					fmt.Printf("Warning: %v\n", err)
				} else {
					fmt.Printf("🔄 Reverted: %s\n", t.Issue.FilePath)
				}
			}
			if err := s.manager.SkipTask(t.ID, reason); err != nil {
				return err
			}
			fmt.Printf("Task skipped: %s\n", t.Issue.Title)
			return nil
		},
	}
	cmd.Flags().StringVarP(&taskID, "id", "i", "", "task ID (default: current in-progress task)")
	cmd.Flags().StringVarP(&reason, "reason", "R", "", "reason for skipping")
	cmd.Flags().BoolVar(&noRevert, "no-revert", false, "skip reverting file changes")
	return cmd
}

func skipBatchCmd() *cobra.Command {
	var reason string
	var noRevert bool

	cmd := &cobra.Command{
		Use:   "skip-batch",
		Short: "Skip all in-progress tasks and revert changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}

			tasks := s.manager.GetInProgressTasks()
			if len(tasks) == 0 {
				return fmt.Errorf("no tasks in progress")
			}

			files, taskIDs := uniqueFilesAndIDs(tasks)
			if !noRevert {
				if err := gitRevert(s.repo, files); err != nil {
					fmt.Printf("Warning: %v\n", err)
				} else {
					for _, f := range files {
						fmt.Printf("🔄 Reverted: %s\n", f)
					}
				}
			}
			if err := s.manager.SkipBatch(taskIDs, reason); err != nil {
				return err
			}
			fmt.Printf("Batch skipped: %d tasks\n", len(tasks))
			return nil
		},
	}
	cmd.Flags().StringVarP(&reason, "reason", "R", "", "reason for skipping")
	cmd.Flags().BoolVar(&noRevert, "no-revert", false, "skip reverting file changes")
	return cmd
}

func resetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset all tasks to pending",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}
			if err := s.manager.Reset(); err != nil {
				return err
			}
			fmt.Println("All tasks reset to pending.")
			return nil
		},
	}
}

func resetProgressCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset-progress",
		Short: "Reset in-progress tasks back to pending",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := newSession()
			if err != nil {
				return err
			}
			count, err := s.manager.ResetInProgress()
			if err != nil {
				return err
			}
			if count == 0 {
				fmt.Println("No in-progress tasks to reset.")
			} else {
				fmt.Printf("Reset %d in-progress tasks to pending.\n", count)
			}
			return nil
		},
	}
}

func printStats(manager *task.Manager) {
	fmt.Println(prompt.GenerateProgressReport(manager.GetStats()))
}

// resolveInProgressTask returns the task with the given ID, or the current in-progress task if id is empty.
func resolveInProgressTask(manager *task.Manager, id string) (*task.Task, error) {
	if id == "" {
		tasks := manager.GetInProgressTasks()
		if len(tasks) == 0 {
			return nil, fmt.Errorf("no task in progress")
		}
		return tasks[0], nil
	}
	t, err := manager.GetTask(id)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func uniqueFilesAndIDs(tasks []*task.Task) (files, taskIDs []string) {
	seen := make(map[string]bool)
	for _, t := range tasks {
		taskIDs = append(taskIDs, t.ID)
		if !seen[t.Issue.FilePath] {
			seen[t.Issue.FilePath] = true
			files = append(files, t.Issue.FilePath)
		}
	}
	sort.Strings(files)
	return files, taskIDs
}

func stageFiles(repo string, files []string) error {
	args := append([]string{"add", "--"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage files: %w\n%s", err, string(output))
	}
	return nil
}

func gitCommit(repo, message string) (string, error) {
	commitCmd := exec.Command("git", "commit", "-m", message)
	commitCmd.Dir = repo
	if output, err := commitCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to commit: %w\n%s", err, string(output))
	}
	hashCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	hashCmd.Dir = repo
	hashOutput, err := hashCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}
	return strings.TrimSpace(string(hashOutput)), nil
}

func gitRevert(repo string, files []string) error {
	args := append([]string{"checkout", "--"}, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout: %w\n%s", err, string(output))
	}
	return nil
}
