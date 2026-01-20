package cascade

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zealllot/dsfix/internal/task"
)

// GenerateFixPrompt generates a prompt for Cascade to fix an issue
func GenerateFixPrompt(t *task.Task, repoPath string) string {
	var sb strings.Builder

	sb.WriteString("## DeepSource Issue Fix Task\n\n")
	sb.WriteString(fmt.Sprintf("**Task ID:** `%s`\n", t.ID))
	sb.WriteString(fmt.Sprintf("**Issue:** %s\n", t.Issue.Title))
	sb.WriteString(fmt.Sprintf("**Category:** %s\n", t.Issue.Category))
	sb.WriteString(fmt.Sprintf("**Severity:** %s\n", t.Issue.Severity))
	sb.WriteString(fmt.Sprintf("**Shortcode:** %s\n", t.Issue.Shortcode))
	sb.WriteString(fmt.Sprintf("**Analyzer:** %s\n\n", t.Issue.Analyzer))

	sb.WriteString("### Location\n")
	sb.WriteString(fmt.Sprintf("- **File:** `%s`\n", t.Issue.FilePath))
	sb.WriteString(fmt.Sprintf("- **Lines:** %d-%d\n\n", t.Issue.BeginLine, t.Issue.EndLine))

	sb.WriteString("### Description\n")
	sb.WriteString(t.Issue.Description)
	sb.WriteString("\n\n")

	if t.Issue.Suggestion != "" {
		sb.WriteString("### Suggestion\n")
		sb.WriteString(t.Issue.Suggestion)
		sb.WriteString("\n\n")
	}

	sb.WriteString("### Instructions\n")
	sb.WriteString("1. Read the file at the specified location\n")
	sb.WriteString("2. Understand the issue and fix it according to best practices\n")
	sb.WriteString("3. Make minimal changes to fix only this specific issue\n")
	sb.WriteString("4. Do not modify unrelated code\n\n")

	sb.WriteString("### ⚠️ IMPORTANT: After Fix\n")
	sb.WriteString("After completing the fix, you MUST:\n")
	sb.WriteString("1. **Run `go build ./...`** to verify the fix compiles without errors\n")
	sb.WriteString("2. If build fails, fix the errors and run build again\n")
	sb.WriteString("3. **Show the changes you made** (what was changed and why)\n")
	sb.WriteString("4. **Show the suggested commit message:**\n")
	sb.WriteString("   ```\n")
	sb.WriteString("   " + t.GenerateCommitMessage() + "\n")
	sb.WriteString("   ```\n")
	sb.WriteString("5. **Ask user to confirm** by saying: \"✅ Build 通过，请确认修复内容，确认后我将自动提交。\"\n")
	sb.WriteString("6. **When user confirms** (says 确认/继续/ok/yes), run: `dsfix complete`\n")
	sb.WriteString("7. **If user wants to skip** (says 跳过/skip), run: `dsfix skip` (this will auto-revert changes)\n\n")

	return sb.String()
}

// GenerateBatchFixPrompt generates a prompt for Cascade to fix multiple issues of the same type
func GenerateBatchFixPrompt(tasks []*task.Task, repoPath string) string {
	if len(tasks) == 0 {
		return "No tasks to fix."
	}

	var sb strings.Builder
	first := tasks[0]

	sb.WriteString("## DeepSource Batch Fix Task\n\n")
	sb.WriteString(fmt.Sprintf("**Issue Type:** %s\n", first.Issue.Title))
	sb.WriteString(fmt.Sprintf("**Shortcode:** %s\n", first.Issue.Shortcode))
	sb.WriteString(fmt.Sprintf("**Category:** %s\n", first.Issue.Category))
	sb.WriteString(fmt.Sprintf("**Total Occurrences:** %d\n\n", len(tasks)))

	sb.WriteString("### Description\n")
	sb.WriteString(first.Issue.Description)
	sb.WriteString("\n\n")

	sb.WriteString("### Locations to Fix\n")
	sb.WriteString("| # | File | Lines | Task ID |\n")
	sb.WriteString("|---|------|-------|--------|\n")
	for i, t := range tasks {
		sb.WriteString(fmt.Sprintf("| %d | `%s` | %d-%d | `%s` |\n",
			i+1, t.Issue.FilePath, t.Issue.BeginLine, t.Issue.EndLine, t.ID[:16]+"..."))
	}
	sb.WriteString("\n")

	sb.WriteString("### ⚠️ CRITICAL: You MUST fix ALL locations\n")
	sb.WriteString(fmt.Sprintf("**There are %d locations to fix. You MUST fix ALL of them, not just some.**\n\n", len(tasks)))
	sb.WriteString("1. Fix **EVERY** occurrence listed in the table above\n")
	sb.WriteString("2. Apply the same fix pattern to each location\n")
	sb.WriteString("3. Make minimal changes to fix only these specific issues\n")
	sb.WriteString("4. Do not modify unrelated code\n")
	sb.WriteString("5. **Do NOT stop until all locations are fixed**\n\n")

	sb.WriteString("### After Fixing ALL Locations\n")
	sb.WriteString("After completing ALL fixes (all rows in the table), you MUST:\n")
	sb.WriteString("1. **Run `go build ./...`** to verify all fixes compile without errors\n")
	sb.WriteString("2. If build fails, fix the errors and run build again\n")
	sb.WriteString("3. **Show a summary of changes** (files modified and what was changed)\n")
	sb.WriteString("4. **Show the suggested commit message:**\n")
	sb.WriteString("   ```\n")
	sb.WriteString(fmt.Sprintf("   fix(%s): %s (%d occurrences)\n", first.Issue.Shortcode, first.Issue.Title, len(tasks)))
	sb.WriteString("   ```\n")
	sb.WriteString("5. **Ask user to confirm** by saying: \"✅ Build 通过，请确认修复内容，确认后我将自动提交。\"\n")
	sb.WriteString("6. **When user confirms** (says 确认/继续/ok/yes), run: `dsfix complete-batch`\n")
	sb.WriteString("7. **If user wants to skip** (says 跳过/skip), run: `dsfix skip-batch` (this will auto-revert all changes)\n\n")

	return sb.String()
}

// GenerateStartPrompt generates a prompt showing available issue types and asking user which to fix
// AI MUST display this table directly to the user in a clear format
func GenerateStartPrompt(tasks []*task.Task) string {
	var sb strings.Builder

	// Count in_progress tasks
	inProgressCount := 0
	for _, t := range tasks {
		if t.Status == task.StatusInProgress {
			inProgressCount++
		}
	}

	// Group by shortcode (include both pending and in_progress)
	groups := make(map[string][]*task.Task)
	for _, t := range tasks {
		if t.Status == task.StatusPending || t.Status == task.StatusInProgress {
			groups[t.Issue.Shortcode] = append(groups[t.Issue.Shortcode], t)
		}
	}

	if len(groups) == 0 {
		return "✅ 没有待处理的任务！所有 issues 已修复完成。"
	}

	// If there are in_progress tasks, show warning
	if inProgressCount > 0 {
		sb.WriteString(fmt.Sprintf("⚠️ **注意:** 有 %d 个任务处于进行中状态，可能是之前中断的。运行 `dsfix reset-progress` 可重置这些任务。\n\n", inProgressCount))
	}

	// Sort by count
	type kv struct {
		Shortcode string
		Tasks     []*task.Task
	}
	var sorted []kv
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

	sb.WriteString("<!-- AI_INSTRUCTION: You MUST display the following task list directly to the user in your response. Do NOT ask user to look at terminal output. Format it nicely with markdown. -->\n\n")

	sb.WriteString(fmt.Sprintf("## 📋 DSFix 任务列表\n\n"))
	sb.WriteString(fmt.Sprintf("**待处理:** %d 个 issues，%d 种类型\n\n", total, len(sorted)))

	sb.WriteString("| 序号 | Shortcode | 问题描述 | 数量 |\n")
	sb.WriteString("|:----:|-----------|----------|-----:|\n")

	// Show top 15 for clarity
	showCount := len(sorted)
	if showCount > 15 {
		showCount = 15
	}

	for i := 0; i < showCount; i++ {
		item := sorted[i]
		title := item.Tasks[0].Issue.Title
		if len(title) > 45 {
			title = title[:42] + "..."
		}
		sb.WriteString(fmt.Sprintf("| **%d** | `%s` | %s | %d |\n",
			i+1, item.Shortcode, title, len(item.Tasks)))
	}

	if len(sorted) > 15 {
		sb.WriteString(fmt.Sprintf("| ... | ... | *还有 %d 种其他类型* | ... |\n", len(sorted)-15))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("**请选择要处理的类型：**\n\n")
	sb.WriteString("回复 **序号** 或 **Shortcode** 开始批量修复，例如：\n")
	sb.WriteString("- `1` - 处理第 1 类问题\n")
	sb.WriteString("- `RVV-B0012` - 处理指定类型\n")
	sb.WriteString("- `1 -l 10` - 只处理 10 个\n\n")

	sb.WriteString("<!-- AI_INSTRUCTION: After displaying this list, wait for user to choose a number or shortcode. When user responds, run: dsfix batch <shortcode> or dsfix batch <shortcode> -l N -->\n")

	return sb.String()
}

// GenerateTaskSummary generates a summary of a task for display
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

// GenerateProgressReport generates a progress report
func GenerateProgressReport(stats map[task.Status]int) string {
	var sb strings.Builder

	total := 0
	for _, count := range stats {
		total += count
	}

	sb.WriteString("## DSFix Progress Report\n\n")
	sb.WriteString(fmt.Sprintf("**Total Tasks:** %d\n\n", total))

	sb.WriteString("| Status | Count |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Pending | %d |\n", stats[task.StatusPending]))
	sb.WriteString(fmt.Sprintf("| In Progress | %d |\n", stats[task.StatusInProgress]))
	sb.WriteString(fmt.Sprintf("| Fixed | %d |\n", stats[task.StatusFixed]))
	sb.WriteString(fmt.Sprintf("| Skipped | %d |\n", stats[task.StatusSkipped]))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", stats[task.StatusFailed]))

	if total > 0 {
		fixed := stats[task.StatusFixed]
		progress := float64(fixed) / float64(total) * 100
		sb.WriteString(fmt.Sprintf("\n**Progress:** %.1f%%\n", progress))
	}

	return sb.String()
}

// GenerateGroupStats generates statistics grouped by shortcode
func GenerateGroupStats(tasks []*task.Task) string {
	var sb strings.Builder

	// Group by shortcode
	groups := make(map[string][]*task.Task)
	for _, t := range tasks {
		if t.Status == task.StatusPending {
			groups[t.Issue.Shortcode] = append(groups[t.Issue.Shortcode], t)
		}
	}

	sb.WriteString("## Pending Issues by Type\n\n")
	sb.WriteString("| Shortcode | Title | Count |\n")
	sb.WriteString("|-----------|-------|-------|\n")

	// Sort by count (descending)
	type kv struct {
		Key   string
		Tasks []*task.Task
	}
	var sorted []kv
	for k, v := range groups {
		sorted = append(sorted, kv{k, v})
	}
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if len(sorted[j].Tasks) > len(sorted[i].Tasks) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	for _, item := range sorted {
		if len(item.Tasks) > 0 {
			sb.WriteString(fmt.Sprintf("| %s | %s | %d |\n",
				item.Key, item.Tasks[0].Issue.Title, len(item.Tasks)))
		}
	}

	return sb.String()
}
