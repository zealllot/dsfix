package cascade

import (
"fmt"
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
	sb.WriteString("4. Do not modify unrelated code\n")
	sb.WriteString("5. **STOP after fixing and wait for user confirmation**\n\n")

	sb.WriteString("### ⚠️ IMPORTANT: After Fix\n")
	sb.WriteString("After you complete the fix, **STOP and ask me to review the changes**.\n")
	sb.WriteString("Do NOT run any commit commands. I will run `dsfix complete` after reviewing.\n\n")

	sb.WriteString("### Suggested Commit Message\n")
	sb.WriteString("```\n")
sb.WriteString(t.GenerateCommitMessage())
sb.WriteString("\n```\n")

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
