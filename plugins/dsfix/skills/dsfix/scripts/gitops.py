"""Thin wrappers around git for staging, committing, and reverting files."""

from __future__ import annotations

import subprocess
from pathlib import Path


class GitError(RuntimeError):
    pass


def _run(args: list[str], cwd: str | Path) -> str:
    p = subprocess.run(args, cwd=str(cwd), capture_output=True, text=True)
    if p.returncode != 0:
        raise GitError(f"git {' '.join(args[1:])} failed:\n{p.stdout}{p.stderr}")
    return p.stdout


def stage(repo: str | Path, files: list[str]) -> None:
    if not files:
        return
    _run(["git", "add", "--", *files], cwd=repo)


def commit(repo: str | Path, message: str) -> str:
    """Create a commit and return the short hash."""
    _run(["git", "commit", "-m", message], cwd=repo)
    return _run(["git", "rev-parse", "--short", "HEAD"], cwd=repo).strip()


def revert(repo: str | Path, files: list[str]) -> None:
    if not files:
        return
    _run(["git", "checkout", "--", *files], cwd=repo)
