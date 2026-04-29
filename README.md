# DSFix

Turn DeepSource issues into per-shortcode batch tasks for an AI assistant (Claude Code, Cursor, Windsurf/Cascade, …) to fix one type at a time, then commit.

## Why

DeepSource often surfaces hundreds of issues. Fixing them one PR at a time is slow; fixing them all at once produces unreviewable diffs. DSFix batches issues by **shortcode** (e.g. `RVV-B0012`) so each commit is one issue type across however many files.

## Install

```bash
go install github.com/zealllot/dsfix/cmd/dsfix@latest
```

Or build from source:

```bash
git clone https://github.com/zealllot/dsfix.git
cd dsfix
go build -o dsfix ./cmd/dsfix
```

## Setup

1. Get a DeepSource Personal Access Token: <https://app.deepsource.io/settings/tokens>
2. Export it (preferred over putting it in YAML):
   ```bash
   export DEEPSOURCE_API_TOKEN="..."
   ```
3. In your project repo:
   ```bash
   dsfix init
   ```
4. Edit `.dsfix.yaml` — set `repository.owner` and `repository.name`, plus any `filter` you want.

## Claude Code integration (recommended)

```bash
dsfix init-claude
```

This drops:
- `.claude/commands/dsfix.md` — a `/dsfix` slash command that walks Claude through the full flow
- `.claude/agents/dsfix-fixer.md` — an optional subagent for delegating large batches

Then in Claude Code, just type `/dsfix`. Claude lists the pending issue types, asks which to handle, applies the fix, runs verify, asks you to confirm, and commits.

## Usage (any AI assistant or manual)

```bash
dsfix sync                      # Fetch issues from DeepSource
dsfix start                     # Show task list grouped by shortcode
dsfix batch SCC-S1039           # Output a fix prompt for one shortcode (also marks tasks in_progress)
                                # → AI applies fixes, runs verify
dsfix complete-batch            # Stage modified files and commit
# or
dsfix skip-batch                # Revert and mark tasks as skipped
```

For one-at-a-time mode: `dsfix next` / `dsfix complete` / `dsfix skip`.

## Configuration reference

```yaml
deepsource:
  api_token: ""                  # Prefer DEEPSOURCE_API_TOKEN env var

repository:
  owner: ""
  name: ""

filter:
  categories: []                 # Bug Risk, Anti-pattern, Security, Performance, Typecheck, Style, Documentation
  severities: []                 # critical, major, minor
  limit:                         # Max issues to fetch; empty = unlimited
  paths_include: []              # Globs; if non-empty, only matching paths kept (e.g. ["internal/**"])
  paths_exclude: []              # Globs; matching paths dropped (e.g. ["vendor/**", "**/*_test.go"])

verify:
  command: ""                    # Run after each fix to verify, e.g. "go build ./..." — empty = AI picks
```

## Commands

| Command | Description |
|---------|-------------|
| `dsfix init` | Create `.dsfix.yaml` template |
| `dsfix init-claude` | Scaffold `.claude/commands/dsfix.md` + `.claude/agents/dsfix-fixer.md` |
| `dsfix sync` | Fetch issues from DeepSource |
| `dsfix status` | Print task counts |
| `dsfix start` | Show task list grouped by shortcode (auto-syncs if empty) |
| `dsfix list` | Same as `start` minus the AI-friendly framing |
| `dsfix batch <shortcode>` | Output fix prompt for one shortcode, mark tasks `in_progress` |
| `dsfix complete-batch` | Stage `in_progress` files, commit, mark `fixed` |
| `dsfix skip-batch` | Revert `in_progress` files, mark `skipped` |
| `dsfix next` | Output a single task prompt |
| `dsfix complete` | Commit current single task |
| `dsfix skip` | Revert and skip current single task |
| `dsfix reset-progress` | Reset stuck `in_progress` tasks back to `pending` |
| `dsfix reset` | Reset all tasks to `pending` |
| `dsfix run` | Legacy interactive terminal-driven mode |

## Files

```
.dsfix.yaml                     # Config (gitignore this — has secrets)
.dsfix/tasks.json               # Task state (gitignore this)
.claude/commands/dsfix.md       # Optional: created by init-claude
.claude/agents/dsfix-fixer.md   # Optional: created by init-claude
```

## License

MIT
