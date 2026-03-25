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
"github.com/zealllot/dsfix/internal/cascade"
"github.com/zealllot/dsfix/internal/deepsource"
"github.com/zealllot/dsfix/internal/task"
)

var (
cfgFile  string
repoPath string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dsfix",
		Short: "DeepSource + Windsurf integration tool",
		Long:  "DSFix automatically fixes DeepSource issues using Windsurf/Cascade",
	}

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is .dsfix.yaml)")
	rootCmd.PersistentFlags().StringVarP(&repoPath, "repo", "r", "", "repository path (default is current directory)")

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(syncCmd())
	rootCmd.AddCommand(runCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(nextCmd())
	rootCmd.AddCommand(completeCmd())
	rootCmd.AddCommand(skipCmd())
	rootCmd.AddCommand(resetCmd())
	rootCmd.AddCommand(resetProgressCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(batchCmd())
	rootCmd.AddCommand(completeBatchCmd())
	rootCmd.AddCommand(skipBatchCmd())
	rootCmd.AddCommand(startCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadConfig() (*config.Config, error) {
	path := cfgFile
	if path == "" {
		if repoPath != "" {
			path = filepath.Join(repoPath, config.DefaultConfigFile)
		} else {
			cwd, _ := os.Getwd()
			path = filepath.Join(cwd, config.DefaultConfigFile)
		}
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
			fmt.Println("Please edit the config file and add your DeepSource API token.")
			return nil
		},
	}
}

func syncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync issues from DeepSource",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			if err := cfg.Validate(); err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)

			filter := &deepsource.IssueFilter{
				Categories: cfg.Filter.Categories,
				Severities: cfg.Filter.Severities,
				Limit:      cfg.Filter.Limit,
			}

			fmt.Println("Fetching issues from DeepSource...")
			count, err := manager.Sync(context.Background(), filter)
			if err != nil {
				return err
			}

			fmt.Printf("Synced %d new issues.\n", count)
			printStats(manager)
			return nil
		},
	}
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run interactive fix process",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			if err := cfg.Validate(); err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)
			executor := cascade.NewExecutor(manager, getRepoPath())

			return executor.RunInteractive()
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)
			printStats(manager)
			return nil
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List pending issues grouped by type",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)
			tasks := manager.GetPendingTasks()

			fmt.Println(cascade.GenerateGroupStats(tasks))
			return nil
		},
	}
}

func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Show task list and let AI guide you through fixing",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)
			tasks := manager.GetPendingTasks()

			// If no pending tasks, try to sync from DeepSource
			if len(tasks) == 0 {
				fmt.Println("No pending tasks. Syncing from DeepSource...")

				if err := cfg.Validate(); err != nil {
					return err
				}

				filter := &deepsource.IssueFilter{
					Categories: cfg.Filter.Categories,
					Severities: cfg.Filter.Severities,
					Limit:      cfg.Filter.Limit,
				}

				count, err := manager.Sync(context.Background(), filter)
				if err != nil {
					return err
				}

				fmt.Printf("Synced %d new issues.\n\n", count)
			}

			fmt.Println(cascade.GenerateStartPrompt(manager.GetAllTasks()))
			return nil
		},
	}
}

func nextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "next",
		Short: "Output the next task for Cascade to fix",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)
			executor := cascade.NewExecutor(manager, getRepoPath())

			t, err := executor.OutputTaskForCascade()
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

			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)
			tasks := manager.GetTasksByShortcode(shortcode)

			if len(tasks) == 0 {
				fmt.Printf("No pending tasks with shortcode: %s\n", shortcode)
				return nil
			}

			// Apply limit only if specified
			if limit > 0 && len(tasks) > limit {
				tasks = tasks[:limit]
			}

			// Mark all as in progress
			var taskIDs []string
			for _, t := range tasks {
				taskIDs = append(taskIDs, t.ID)
			}
			if err := manager.StartBatch(taskIDs); err != nil {
				return err
			}

			// Output batch prompt
			fmt.Println(cascade.GenerateBatchFixPrompt(tasks, getRepoPath()))

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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)

			// Find the in-progress task if no ID specified
			if taskID == "" {
				tasks := store.GetByStatus(task.StatusInProgress)
				if len(tasks) == 0 {
					return fmt.Errorf("no task in progress")
				}
				taskID = tasks[0].ID
			}

			t, ok := store.Get(taskID)
			if !ok {
				return fmt.Errorf("task not found: %s", taskID)
			}

			if commitMsg == "" {
				commitMsg = t.GenerateCommitMessage()
			}

			var commitHash string
			if !noCommit {
				// Stage the file
				stageCmd := exec.Command("git", "add", t.Issue.FilePath)
				stageCmd.Dir = getRepoPath()
				if output, err := stageCmd.CombinedOutput(); err != nil {
					return fmt.Errorf("failed to stage file: %w\n%s", err, string(output))
				}

				// Create commit
				gitCommitCmd := exec.Command("git", "commit", "-m", commitMsg)
				gitCommitCmd.Dir = getRepoPath()
				if output, err := gitCommitCmd.CombinedOutput(); err != nil {
					return fmt.Errorf("failed to commit: %w\n%s", err, string(output))
				}

				// Get commit hash
				hashCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
				hashCmd.Dir = getRepoPath()
				hashOutput, err := hashCmd.Output()
				if err != nil {
					return fmt.Errorf("failed to get commit hash: %w", err)
				}
				commitHash = strings.TrimSpace(string(hashOutput))
				fmt.Printf("✅ Committed: %s\n", commitHash)
			} else {
				commitHash = "no-commit"
			}

			if err := manager.CompleteTask(taskID, commitHash, commitMsg); err != nil {
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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)

			tasks := manager.GetInProgressTasks()
			if len(tasks) == 0 {
				return fmt.Errorf("no tasks in progress")
			}

			// Collect unique files and task IDs
			filesMap := make(map[string]bool)
			var taskIDs []string
			for _, t := range tasks {
				filesMap[t.Issue.FilePath] = true
				taskIDs = append(taskIDs, t.ID)
			}

			var files []string
			for f := range filesMap {
				files = append(files, f)
			}
			sort.Strings(files)

			// Generate commit message if not provided
			if commitMsg == "" {
				first := tasks[0]
				commitMsg = fmt.Sprintf("fix(%s): %s (%d occurrences)", first.Issue.Shortcode, first.Issue.Title, len(tasks))
			}

			var commitHash string
			if !noCommit {
				// Stage all files
				for _, f := range files {
					stageCmd := exec.Command("git", "add", f)
					stageCmd.Dir = getRepoPath()
					if output, err := stageCmd.CombinedOutput(); err != nil {
						fmt.Printf("Warning: failed to stage %s: %v\n%s", f, err, string(output))
					}
				}

				// Create commit
				gitCommitCmd := exec.Command("git", "commit", "-m", commitMsg)
				gitCommitCmd.Dir = getRepoPath()
				if output, err := gitCommitCmd.CombinedOutput(); err != nil {
					return fmt.Errorf("failed to commit: %w\n%s", err, string(output))
				}

				// Get commit hash
				hashCmd := exec.Command("git", "rev-parse", "--short", "HEAD")
				hashCmd.Dir = getRepoPath()
				hashOutput, err := hashCmd.Output()
				if err != nil {
					return fmt.Errorf("failed to get commit hash: %w", err)
				}
				commitHash = strings.TrimSpace(string(hashOutput))
				fmt.Printf("✅ Committed: %s\n", commitHash)
			} else {
				commitHash = "no-commit"
			}

			if err := manager.CompleteBatch(taskIDs, commitHash, commitMsg); err != nil {
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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)

			// Find the in-progress task if no ID specified
			if taskID == "" {
				tasks := store.GetByStatus(task.StatusInProgress)
				if len(tasks) == 0 {
					return fmt.Errorf("no task in progress")
				}
				taskID = tasks[0].ID
			}

			t, ok := store.Get(taskID)
			if !ok {
				return fmt.Errorf("task not found: %s", taskID)
			}

			// Revert changes to the file
			if !noRevert {
				revertCmd := exec.Command("git", "checkout", "--", t.Issue.FilePath)
				revertCmd.Dir = getRepoPath()
				if output, err := revertCmd.CombinedOutput(); err != nil {
					fmt.Printf("Warning: failed to revert file %s: %v\n%s", t.Issue.FilePath, err, string(output))
				} else {
					fmt.Printf("🔄 Reverted: %s\n", t.Issue.FilePath)
				}
			}

			if err := manager.SkipTask(taskID, reason); err != nil {
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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)

			tasks := manager.GetInProgressTasks()
			if len(tasks) == 0 {
				return fmt.Errorf("no tasks in progress")
			}

			// Collect unique files and task IDs
			filesMap := make(map[string]bool)
			var taskIDs []string
			for _, t := range tasks {
				filesMap[t.Issue.FilePath] = true
				taskIDs = append(taskIDs, t.ID)
			}

			// Revert all files
			if !noRevert {
				for f := range filesMap {
					revertCmd := exec.Command("git", "checkout", "--", f)
					revertCmd.Dir = getRepoPath()
					if output, err := revertCmd.CombinedOutput(); err != nil {
						fmt.Printf("Warning: failed to revert %s: %v\n%s", f, err, string(output))
					} else {
						fmt.Printf("🔄 Reverted: %s\n", f)
					}
				}
			}

			if err := manager.SkipBatch(taskIDs, reason); err != nil {
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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)

			if err := manager.Reset(); err != nil {
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
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			client := deepsource.NewClient(cfg.DeepSource.APIToken)

			store, err := task.NewStore(getRepoPath())
			if err != nil {
				return err
			}

			manager := task.NewManager(store, client, cfg.Repository.Owner, cfg.Repository.Name)

			count, err := manager.ResetInProgress()
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
	fmt.Println(cascade.GenerateProgressReport(manager.GetStats()))
}
