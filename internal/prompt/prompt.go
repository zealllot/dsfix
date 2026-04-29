package prompt

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zealllot/dsfix/internal/task"
)

// verifyLine returns the verify instruction line, or empty if no verify command is set.
func verifyLine(verifyCmd string) string {
	if verifyCmd == "" {
		return "Run the project's standard build/lint command to verify the fix compiles cleanly."
	}
	return fmt.Sprintf("Run `%s` to verify the fix compiles cleanly.", verifyCmd)
}

// GenerateFixPrompt generates a prompt to fix a single issue.
func GenerateFixPrompt(t *task.Task, verifyCmd string) string {
	var sb strings.Builder

	sb.WriteString("## DeepSource Issue Fix Task\n\n")
	fmt.Fprintf(&sb, "**Task ID:** `%s`\n", t.ID)
	fmt.Fprintf(&sb, "**Issue:** %s\n", t.Issue.Title)
	fmt.Fprintf(&sb, "**Category:** %s\n", t.Issue.Category)
	fmt.Fprintf(&sb, "**Severity:** %s\n", t.Issue.Severity)
	fmt.Fprintf(&sb, "**Shortcode:** %s\n", t.Issue.Shortcode)
	fmt.Fprintf(&sb, "**Analyzer:** %s\n\n", t.Issue.Analyzer)

	sb.WriteString("### Location\n")
	fmt.Fprintf(&sb, "- **File:** `%s`\n", t.Issue.FilePath)
	fmt.Fprintf(&sb, "- **Lines:** %d-%d\n\n", t.Issue.BeginLine, t.Issue.EndLine)

	sb.WriteString("### Description\n")
	sb.WriteString(t.Issue.Description)
	sb.WriteString("\n\n")

	if t.Issue.Suggestion != "" {
		sb.WriteString("### Suggestion\n")
		sb.WriteString(t.Issue.Suggestion)
		sb.WriteString("\n\n")
	}

	sb.WriteString("### Workflow\n")
	sb.WriteString("1. Read the file at the location above and fix only this issue (minimal change).\n")
	fmt.Fprintf(&sb, "2. %s\n", verifyLine(verifyCmd))
	sb.WriteString("3. Show the diff and the suggested commit message:\n")
	fmt.Fprintf(&sb, "   ```\n   %s\n   ```\n", t.GenerateCommitMessage())
	sb.WriteString("4. Ask the user to confirm.\n")
	sb.WriteString("5. On confirm: run `dsfix complete`. On reject: run `dsfix skip`.\n\n")

	return sb.String()
}

// GenerateBatchFixPrompt generates a prompt to fix multiple issues of the same type.
func GenerateBatchFixPrompt(tasks []*task.Task, verifyCmd string) string {
	if len(tasks) == 0 {
		return "No tasks to fix."
	}

	var sb strings.Builder
	first := tasks[0]

	sb.WriteString("## DeepSource Batch Fix Task\n\n")
	fmt.Fprintf(&sb, "**Issue Type:** %s\n", first.Issue.Title)
	fmt.Fprintf(&sb, "**Shortcode:** %s\n", first.Issue.Shortcode)
	fmt.Fprintf(&sb, "**Category:** %s\n", first.Issue.Category)
	fmt.Fprintf(&sb, "**Total Occurrences:** %d\n\n", len(tasks))

	sb.WriteString("### Description\n")
	sb.WriteString(first.Issue.Description)
	sb.WriteString("\n\n")

	sb.WriteString("### Locations to Fix\n")
	sb.WriteString("Tasks are sorted by file ascending, line descending — fixing in this order avoids line-number shifts within a file.\n\n")
	sb.WriteString("| # | File | Lines |\n")
	sb.WriteString("|---|------|-------|\n")
	for i, t := range tasks {
		fmt.Fprintf(&sb, "| %d | `%s` | %d-%d |\n",
			i+1, t.Issue.FilePath, t.Issue.BeginLine, t.Issue.EndLine)
	}
	sb.WriteString("\n")

	sb.WriteString("### Workflow\n")
	fmt.Fprintf(&sb, "1. Apply the same fix pattern to **every** location above (%d total). Do not stop partway.\n", len(tasks))
	sb.WriteString("2. Make minimal changes — fix only this issue, do not refactor unrelated code.\n")
	fmt.Fprintf(&sb, "3. %s\n", verifyLine(verifyCmd))
	sb.WriteString("4. Show a brief summary of changes (files touched, what changed).\n")
	fmt.Fprintf(&sb, "5. Suggested commit message:\n   ```\n   fix(%s): %s (%d occurrences)\n   ```\n",
		first.Issue.Shortcode, first.Issue.Title, len(tasks))
	sb.WriteString("6. Ask the user to confirm.\n")
	sb.WriteString("7. On confirm: run `dsfix complete-batch`. On reject: run `dsfix skip-batch`.\n\n")

	return sb.String()
}

// GenerateStartPrompt generates a prompt listing pending issue types for the user to choose from.
func GenerateStartPrompt(tasks []*task.Task) string {
	var sb strings.Builder

	inProgressCount := 0
	groups := make(map[string][]*task.Task)
	for _, t := range tasks {
		if t.Status == task.StatusInProgress {
			inProgressCount++
		}
		if t.Status == task.StatusPending || t.Status == task.StatusInProgress {
			groups[t.Issue.Shortcode] = append(groups[t.Issue.Shortcode], t)
		}
	}

	if len(groups) == 0 {
		return "✅ 没有待处理的任务！所有 issues 已修复完成。"
	}

	if inProgressCount > 0 {
		fmt.Fprintf(&sb, "⚠️ 有 %d 个任务处于进行中状态（可能是之前中断的）。运行 `dsfix reset-progress` 可重置。\n\n", inProgressCount)
	}

	type kv struct {
		Shortcode string
		Tasks     []*task.Task
	}
	sorted := make([]kv, 0, len(groups))
	for k, v := range groups {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Tasks) > len(sorted[j].Tasks)
	})

	total := 0
	for _, item := range sorted {
		total += len(item.Tasks)
	}

	sb.WriteString("## 📋 DSFix 任务列表\n\n")
	fmt.Fprintf(&sb, "**待处理:** %d 个 issues，%d 种类型\n\n", total, len(sorted))

	sb.WriteString("| 序号 | Shortcode | 问题描述 | 数量 |\n")
	sb.WriteString("|:----:|-----------|----------|-----:|\n")

	showCount := min(len(sorted), 15)
	for i := 0; i < showCount; i++ {
		item := sorted[i]
		title := item.Tasks[0].Issue.Title
		if len(title) > 45 {
			title = title[:42] + "..."
		}
		fmt.Fprintf(&sb, "| **%d** | `%s` | %s | %d |\n",
			i+1, item.Shortcode, title, len(item.Tasks))
	}

	if len(sorted) > 15 {
		fmt.Fprintf(&sb, "| ... | ... | *还有 %d 种其他类型* | ... |\n", len(sorted)-15)
	}

	sb.WriteString("\n请选择序号或 shortcode，然后运行 `dsfix batch <shortcode>` 开始批量修复。\n")

	return sb.String()
}

// GenerateTaskSummary generates a one-line summary of a task.
func GenerateTaskSummary(t *task.Task) string {
	return fmt.Sprintf("[%s] %s @ %s:%d-%d (%s)",
		t.Issue.Severity,
		t.Issue.Title,
		t.Issue.FilePath,
		t.Issue.BeginLine,
		t.Issue.EndLine,
		t.Issue.Shortcode,
	)
}

// GenerateProgressReport generates a progress report from status counts.
func GenerateProgressReport(stats map[task.Status]int) string {
	var sb strings.Builder

	total := 0
	for _, count := range stats {
		total += count
	}

	sb.WriteString("## DSFix Progress Report\n\n")
	fmt.Fprintf(&sb, "**Total Tasks:** %d\n\n", total)

	sb.WriteString("| Status | Count |\n")
	sb.WriteString("|--------|-------|\n")
	fmt.Fprintf(&sb, "| Pending | %d |\n", stats[task.StatusPending])
	fmt.Fprintf(&sb, "| In Progress | %d |\n", stats[task.StatusInProgress])
	fmt.Fprintf(&sb, "| Fixed | %d |\n", stats[task.StatusFixed])
	fmt.Fprintf(&sb, "| Skipped | %d |\n", stats[task.StatusSkipped])
	fmt.Fprintf(&sb, "| Failed | %d |\n", stats[task.StatusFailed])

	if total > 0 {
		fixed := stats[task.StatusFixed]
		progress := float64(fixed) / float64(total) * 100
		fmt.Fprintf(&sb, "\n**Progress:** %.1f%%\n", progress)
	}

	return sb.String()
}

// GenerateGroupStats generates statistics grouped by shortcode for pending tasks.
func GenerateGroupStats(tasks []*task.Task) string {
	var sb strings.Builder

	groups := make(map[string][]*task.Task)
	for _, t := range tasks {
		if t.Status == task.StatusPending {
			groups[t.Issue.Shortcode] = append(groups[t.Issue.Shortcode], t)
		}
	}

	sb.WriteString("## Pending Issues by Type\n\n")
	sb.WriteString("| Shortcode | Title | Count |\n")
	sb.WriteString("|-----------|-------|-------|\n")

	type kv struct {
		Key   string
		Tasks []*task.Task
	}
	sorted := make([]kv, 0, len(groups))
	for k, v := range groups {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Tasks) > len(sorted[j].Tasks)
	})

	for _, item := range sorted {
		if len(item.Tasks) > 0 {
			fmt.Fprintf(&sb, "| %s | %s | %d |\n",
				item.Key, item.Tasks[0].Issue.Title, len(item.Tasks))
		}
	}

	return sb.String()
}
