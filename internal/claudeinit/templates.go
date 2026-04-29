// Package claudeinit scaffolds Claude Code integration files (slash commands, subagents)
// into a target repository.
package claudeinit

import (
	"fmt"
	"os"
	"path/filepath"
)

// File is a single scaffolded file with its target relative path and content.
type File struct {
	Path    string // relative to repo root, e.g. ".claude/commands/dsfix.md"
	Content string
}

// Files returns all integration files dsfix scaffolds.
func Files() []File {
	return []File{
		{Path: ".claude/commands/dsfix.md", Content: dsfixCommand},
		{Path: ".claude/agents/dsfix-fixer.md", Content: dsfixFixerAgent},
	}
}

// Scaffold writes integration files into repoPath. If overwrite is false, existing
// files are skipped (and reported in the returned skipped slice).
func Scaffold(repoPath string, overwrite bool) (written, skipped []string, err error) {
	for _, f := range Files() {
		full := filepath.Join(repoPath, f.Path)
		if _, statErr := os.Stat(full); statErr == nil && !overwrite {
			skipped = append(skipped, f.Path)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return written, skipped, fmt.Errorf("mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(f.Content), 0644); err != nil {
			return written, skipped, fmt.Errorf("write %s: %w", full, err)
		}
		written = append(written, f.Path)
	}
	return written, skipped, nil
}

const dsfixCommand = `---
description: Run the DSFix workflow — list pending DeepSource issue types, pick one, and batch-fix.
argument-hint: "[shortcode]"
allowed-tools: Bash(dsfix *), Bash(git *), Read, Edit, Grep, Glob
---

You are running the DSFix workflow.

## Step 1 — Show the task list (skip if user already gave a shortcode)

If $ARGUMENTS is empty:
1. Run ` + "`dsfix start`" + ` and display the task table to the user.
2. Wait for the user to choose a number or shortcode. Then continue with that shortcode.

If $ARGUMENTS contains a shortcode, use it directly.

## Step 2 — Batch fix

1. Run ` + "`dsfix batch <shortcode>`" + `. The output describes every occurrence to fix and the verify command.
2. Apply the fix to every listed location. Make minimal changes. Do not refactor unrelated code.
3. Run the verify command from the dsfix output (or, if it says "let the AI choose", pick the project's standard build/lint).
4. If verification fails, fix the errors and re-verify.

## Step 3 — Confirm and commit

1. Show the user a brief summary: files touched, what changed, count of occurrences.
2. Show the suggested commit message from the dsfix output.
3. Ask the user to confirm.
4. On confirm: run ` + "`dsfix complete-batch`" + `. On reject: run ` + "`dsfix skip-batch`" + ` (auto-reverts).

## Notes

- The store at .dsfix/tasks.json tracks which occurrences are pending/in_progress/fixed.
- If a previous run was interrupted, ` + "`dsfix reset-progress`" + ` resets stuck in_progress tasks.
- For very large batches, consider delegating Step 2 to the dsfix-fixer subagent.
`

const dsfixFixerAgent = `---
name: dsfix-fixer
description: Apply DeepSource fixes for one shortcode (a batch of occurrences). Use when the main agent wants to delegate a fix-and-verify loop to a focused subagent.
tools: Read, Edit, Grep, Glob, Bash
---

You are a focused worker that fixes one batch of DeepSource issues.

You will be given a shortcode. Your job:

1. Run ` + "`dsfix batch <shortcode>`" + ` to get the list of occurrences and fix instructions.
2. Apply the fix to **every** occurrence. The output is sorted file-asc, line-desc — keep that order to avoid line-number shifts.
3. Make minimal changes. Do not refactor unrelated code. Do not add tests unless the issue is about missing tests.
4. Run the verify command from the dsfix output. Fix any failures.
5. Return a summary to the main agent:
   - shortcode and total occurrences fixed
   - files touched (paths only)
   - one-line description of the fix pattern applied
   - verification status

Do NOT run ` + "`dsfix complete-batch`" + ` or ` + "`dsfix skip-batch`" + ` — leave that decision to the main agent + user.
`
