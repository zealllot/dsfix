#!/usr/bin/env python3
"""DSFix CLI — pure-Python equivalent of the legacy Go binary.

Subcommands mirror the Go version:
  init          create .dsfix.yaml template
  sync          fetch issues from DeepSource
  status        show task counts
  list          list pending issues by shortcode
  start         show task list (prompt for AI)
  next          output next single task prompt
  batch         output prompt for one shortcode (mark in_progress)
  complete      commit + mark single task fixed
  complete-batch commit + mark batch fixed
  skip          revert + mark single task skipped
  skip-batch    revert + mark batch skipped
  reset         reset all to pending
  reset-progress reset only in_progress to pending
"""

from __future__ import annotations

import argparse
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))

import api
import config
import gitops
import prompts
import store


def _repo_path(args: argparse.Namespace) -> str:
    return args.repo or os.getcwd()


def _setup(args: argparse.Namespace, *, require_token: bool = False):
    cfg = config.load(_repo_path(args))
    if require_token:
        cfg.validate()
    s = store.Store(_repo_path(args))
    return cfg, s


def _filter(cfg: config.Config) -> api.IssueFilter:
    return api.IssueFilter(
        categories=cfg.filter.categories,
        severities=cfg.filter.severities,
        limit=cfg.filter.limit,
        paths_include=cfg.filter.paths_include,
        paths_exclude=cfg.filter.paths_exclude,
    )


def _print_stats(s: store.Store) -> None:
    print(prompts.progress_report(s.counts()))


def _unique_files_and_ids(tasks: list[store.Task]) -> tuple[list[str], list[str]]:
    seen: set[str] = set()
    files: list[str] = []
    ids: list[str] = []
    for t in tasks:
        ids.append(t.id)
        if t.issue.file_path not in seen:
            seen.add(t.issue.file_path)
            files.append(t.issue.file_path)
    files.sort()
    return files, ids


# --- subcommands ---

def cmd_init(args: argparse.Namespace) -> int:
    repo = _repo_path(args)
    try:
        path = config.write_template(repo)
    except FileExistsError as e:
        print(str(e), file=sys.stderr)
        return 1
    print(f"Created config file: {path}")
    print("Edit it (or set DEEPSOURCE_API_TOKEN env var) and add your DeepSource API token.")
    return 0


def cmd_sync(args: argparse.Namespace) -> int:
    cfg, s = _setup(args, require_token=True)
    print("Fetching issues from DeepSource...")
    client = api.Client(cfg.api_token)
    issues = client.fetch_issues(cfg.owner, cfg.name, _filter(cfg))
    added = s.add_if_new(issues)
    s.save()
    print(f"Synced {added} new issues.")
    _print_stats(s)
    return 0


def cmd_status(args: argparse.Namespace) -> int:
    _, s = _setup(args)
    _print_stats(s)
    return 0


def cmd_list(args: argparse.Namespace) -> int:
    _, s = _setup(args)
    print(prompts.group_stats(s.pending_and_in_progress()))
    return 0


def cmd_start(args: argparse.Namespace) -> int:
    cfg, s = _setup(args)
    if not s.pending_and_in_progress():
        print("No pending tasks. Syncing from DeepSource...")
        cfg.validate()
        client = api.Client(cfg.api_token)
        issues = client.fetch_issues(cfg.owner, cfg.name, _filter(cfg))
        added = s.add_if_new(issues)
        s.save()
        print(f"Synced {added} new issues.\n")
    print(prompts.start_prompt(s.all()))
    return 0


def cmd_next(args: argparse.Namespace) -> int:
    cfg, s = _setup(args)
    pending = s.pending_and_in_progress()
    if not pending:
        print("No pending tasks.")
        return 0
    in_progress = s.by_status(store.Status.IN_PROGRESS)
    t = in_progress[0] if in_progress else pending[0]
    s.mark_in_progress([t.id])
    s.save()
    print(prompts.fix_prompt(t, cfg.verify_command))
    return 0


def cmd_batch(args: argparse.Namespace) -> int:
    cfg, s = _setup(args)
    tasks = s.by_shortcode(args.shortcode)
    if not tasks:
        print(f"No pending tasks with shortcode: {args.shortcode}")
        return 0
    if args.limit and args.limit > 0:
        tasks = tasks[: args.limit]
    s.mark_in_progress([t.id for t in tasks])
    s.save()
    print(prompts.batch_fix_prompt(tasks, cfg.verify_command))
    return 0


def _resolve_in_progress(s: store.Store, task_id: str | None) -> store.Task:
    if task_id:
        t = s.get(task_id)
        if t is None:
            raise SystemExit(f"task not found: {task_id}")
        return t
    in_progress = s.by_status(store.Status.IN_PROGRESS)
    if not in_progress:
        raise SystemExit("no task in progress")
    return in_progress[0]


def cmd_complete(args: argparse.Namespace) -> int:
    _, s = _setup(args)
    repo = _repo_path(args)
    t = _resolve_in_progress(s, args.id)
    msg = args.message or t.commit_message()
    commit_hash = "no-commit"
    if not args.no_commit:
        gitops.stage(repo, [t.issue.file_path])
        commit_hash = gitops.commit(repo, msg)
        print(f"✅ Committed: {commit_hash}")
    s.mark_fixed([t.id], commit_hash, msg)
    s.save()
    print(f"Task completed: {t.issue.title}")
    print(f"Commit message: {msg}")
    return 0


def cmd_complete_batch(args: argparse.Namespace) -> int:
    _, s = _setup(args)
    repo = _repo_path(args)
    tasks = s.by_status(store.Status.IN_PROGRESS)
    if not tasks:
        print("no tasks in progress", file=sys.stderr)
        return 1
    files, ids = _unique_files_and_ids(tasks)
    first = tasks[0]
    msg = args.message or f"fix({first.issue.shortcode}): {first.issue.title} ({len(tasks)} occurrences)"
    commit_hash = "no-commit"
    if not args.no_commit:
        gitops.stage(repo, files)
        commit_hash = gitops.commit(repo, msg)
        print(f"✅ Committed: {commit_hash}")
    s.mark_fixed(ids, commit_hash, msg)
    s.save()
    print(f"Batch completed: {len(tasks)} tasks")
    print(f"Files modified: {len(files)}")
    print(f"Commit message: {msg}")
    return 0


def cmd_skip(args: argparse.Namespace) -> int:
    _, s = _setup(args)
    repo = _repo_path(args)
    t = _resolve_in_progress(s, args.id)
    if not args.no_revert:
        try:
            gitops.revert(repo, [t.issue.file_path])
            print(f"🔄 Reverted: {t.issue.file_path}")
        except gitops.GitError as e:
            print(f"Warning: {e}", file=sys.stderr)
    s.mark_skipped([t.id], args.reason or "")
    s.save()
    print(f"Task skipped: {t.issue.title}")
    return 0


def cmd_skip_batch(args: argparse.Namespace) -> int:
    _, s = _setup(args)
    repo = _repo_path(args)
    tasks = s.by_status(store.Status.IN_PROGRESS)
    if not tasks:
        print("no tasks in progress", file=sys.stderr)
        return 1
    files, ids = _unique_files_and_ids(tasks)
    if not args.no_revert:
        try:
            gitops.revert(repo, files)
            for f in files:
                print(f"🔄 Reverted: {f}")
        except gitops.GitError as e:
            print(f"Warning: {e}", file=sys.stderr)
    s.mark_skipped(ids, args.reason or "")
    s.save()
    print(f"Batch skipped: {len(tasks)} tasks")
    return 0


def cmd_reset(args: argparse.Namespace) -> int:
    _, s = _setup(args)
    s.reset_all()
    s.save()
    print("All tasks reset to pending.")
    return 0


def cmd_reset_progress(args: argparse.Namespace) -> int:
    _, s = _setup(args)
    n = s.reset_in_progress()
    s.save()
    if n == 0:
        print("No in-progress tasks to reset.")
    else:
        print(f"Reset {n} in-progress tasks to pending.")
    return 0


# --- argparse ---

def build_parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(prog="dsfix", description="DeepSource fix workflow.")
    p.add_argument("-r", "--repo", default="", help="repository path (default: cwd)")
    sub = p.add_subparsers(dest="cmd", required=True)

    sub.add_parser("init", help="create .dsfix.yaml template").set_defaults(func=cmd_init)
    sub.add_parser("sync", help="fetch issues from DeepSource").set_defaults(func=cmd_sync)
    sub.add_parser("status", help="show task counts").set_defaults(func=cmd_status)
    sub.add_parser("list", help="list pending issues by shortcode").set_defaults(func=cmd_list)
    sub.add_parser("start", help="show task list and let the AI guide the flow").set_defaults(func=cmd_start)
    sub.add_parser("next", help="output the next single task prompt").set_defaults(func=cmd_next)

    pb = sub.add_parser("batch", help="output prompt for one shortcode")
    pb.add_argument("shortcode")
    pb.add_argument("-l", "--limit", type=int, default=0, help="limit number of tasks (0 = all)")
    pb.set_defaults(func=cmd_batch)

    pc = sub.add_parser("complete", help="commit and mark single task fixed")
    pc.add_argument("-i", "--id", default="", help="task ID (default: current in-progress task)")
    pc.add_argument("-m", "--message", default="")
    pc.add_argument("--no-commit", action="store_true")
    pc.set_defaults(func=cmd_complete)

    pcb = sub.add_parser("complete-batch", help="commit and mark batch fixed")
    pcb.add_argument("-m", "--message", default="")
    pcb.add_argument("--no-commit", action="store_true")
    pcb.set_defaults(func=cmd_complete_batch)

    ps = sub.add_parser("skip", help="revert and mark single task skipped")
    ps.add_argument("-i", "--id", default="")
    ps.add_argument("-R", "--reason", default="")
    ps.add_argument("--no-revert", action="store_true")
    ps.set_defaults(func=cmd_skip)

    psb = sub.add_parser("skip-batch", help="revert and mark batch skipped")
    psb.add_argument("-R", "--reason", default="")
    psb.add_argument("--no-revert", action="store_true")
    psb.set_defaults(func=cmd_skip_batch)

    sub.add_parser("reset", help="reset all tasks to pending").set_defaults(func=cmd_reset)
    sub.add_parser("reset-progress", help="reset in-progress tasks to pending").set_defaults(func=cmd_reset_progress)

    return p


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
