# DSFix

A Claude Code plugin that batches DeepSource code-quality issues by shortcode and walks Claude through fixing each type with one commit per type.

中文版：[README_zh.md](./README_zh.md)

## Why

DeepSource often surfaces hundreds of issues. Fixing them one PR at a time is slow; fixing them all at once produces unreviewable diffs. DSFix groups issues by **shortcode** (e.g. `RVV-B0012`) so each commit is one issue type across however many files.

DSFix is delivered as a Claude Code plugin: install it once, and Claude can drive the whole sync → fix → commit loop with you in plain conversation.

## Install

```
/plugin marketplace add zealllot/dsfix
/plugin install dsfix@zealllot-tools
```

Requires Python 3 and `PyYAML`. If pip refuses on a PEP 668 system (modern macOS / Ubuntu 24+), install with:

```
pipx install PyYAML
# or
python3 -m pip install --break-system-packages PyYAML
```

## Setup (per repo)

1. Get a DeepSource Personal Access Token: <https://app.deepsource.io/settings/tokens> (each user gets their own — don't share).
2. Export it:
   ```bash
   export DEEPSOURCE_API_TOKEN="..."
   ```
3. In your project, ask Claude:
   > Run dsfix init

   That writes a `.dsfix.yaml` template — fill in `repository.owner` and `repository.name`, plus any filters you want.

## Use

Just talk to Claude:

> Fix the DeepSource issues in this repo

Or invoke explicitly:

> /dsfix

Claude will:
1. Run `dsfix start` and show you the table of pending issue types.
2. Wait for you to pick a number or shortcode.
3. Run `dsfix batch <shortcode>`, apply the same fix pattern to every occurrence.
4. Run the verify command (configured or auto-picked).
5. Show you a summary and the suggested commit message.
6. On confirm: `dsfix complete-batch` (auto-stage + commit). On reject: `dsfix skip-batch` (auto-revert).

Repeat until you're done.

## Configuration

`.dsfix.yaml` in your repo root. Full reference: [`plugins/dsfix/skills/dsfix/references/config.md`](./plugins/dsfix/skills/dsfix/references/config.md).

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
  paths_include: []              # Globs with ** support; e.g. ["internal/**"]
  paths_exclude: []              # e.g. ["vendor/**", "**/*_test.go"]

verify:
  command: ""                    # e.g. "go build ./..."; empty = AI picks
```

## Files

- `.claude-plugin/marketplace.json` — registers this repo as a Claude Code marketplace
- `plugins/dsfix/` — the plugin itself
  - `.claude-plugin/plugin.json` — plugin manifest
  - `skills/dsfix/SKILL.md` — main skill entrypoint
  - `skills/dsfix/references/` — workflow / config / api-token detail
  - `skills/dsfix/scripts/` — Python implementation (no Go runtime needed)
  - `commands/dsfix.md` — `/dsfix` slash command
  - `agents/dsfix-fixer.md` — subagent for delegating large batches

## Architecture

`dsfix.py` exposes a CLI mirroring the original Go binary. Claude Code calls it via Bash. The skill description triggers automatic activation when DeepSource is mentioned; the slash command is for explicit invocation. The subagent is optional for very large batches.

## Migration from the Go version

The Go binary at `v0-go-final` (and earlier tags) still works:

```bash
git checkout v0-go-final
go install ./cmd/dsfix
```

`main` after this commit is plugin-only. The Go binary is no longer maintained.

## License

MIT
