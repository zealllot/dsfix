"""Task store: persistence + status queries for DeepSource fix tasks.

Tasks are kept in <repo>/.dsfix/tasks.json. Each task corresponds to a single
DeepSource issue occurrence and tracks its fix status.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field, asdict
from datetime import datetime, timezone
from enum import Enum
from pathlib import Path
from typing import Iterable

STORE_DIR = ".dsfix"
STORE_FILE = "tasks.json"


class Status(str, Enum):
    PENDING = "pending"
    IN_PROGRESS = "in_progress"
    FIXED = "fixed"
    SKIPPED = "skipped"
    FAILED = "failed"


@dataclass
class Issue:
    id: str = ""
    title: str = ""
    category: str = ""
    shortcode: str = ""
    severity: str = ""
    file_path: str = ""
    begin_line: int = 0
    end_line: int = 0
    description: str = ""
    suggestion: str = ""
    analyzer: str = ""


@dataclass
class Task:
    id: str
    issue: Issue
    status: Status = Status.PENDING
    commit_hash: str = ""
    commit_msg: str = ""
    fixed_at: str = ""
    error_msg: str = ""
    created_at: str = ""
    updated_at: str = ""

    @classmethod
    def new(cls, issue: Issue) -> "Task":
        now = _now()
        return cls(id=issue.id, issue=issue, status=Status.PENDING, created_at=now, updated_at=now)

    def commit_message(self) -> str:
        return f"fix({self.issue.shortcode}): {self.issue.title}"


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


class Store:
    def __init__(self, repo_path: str | Path):
        self.repo_path = Path(repo_path)
        self.dir = self.repo_path / STORE_DIR
        self.path = self.dir / STORE_FILE
        self.tasks: dict[str, Task] = {}
        self.dir.mkdir(parents=True, exist_ok=True)
        if self.path.exists():
            self._load()

    # --- IO ---

    def _load(self) -> None:
        with self.path.open("r", encoding="utf-8") as f:
            data = json.load(f)
        self.tasks = {}
        for raw in data.get("tasks", []):
            issue = Issue(**raw.get("issue", {}))
            t = Task(
                id=raw["id"],
                issue=issue,
                status=Status(raw.get("status", Status.PENDING)),
                commit_hash=raw.get("commit_hash", ""),
                commit_msg=raw.get("commit_msg", ""),
                fixed_at=raw.get("fixed_at", ""),
                error_msg=raw.get("error_msg", ""),
                created_at=raw.get("created_at", ""),
                updated_at=raw.get("updated_at", ""),
            )
            self.tasks[t.id] = t

    def save(self) -> None:
        out = {
            "version": "1.0",
            "tasks": [
                {**asdict(t), "issue": asdict(t.issue), "status": t.status.value}
                for t in self.tasks.values()
            ],
        }
        with self.path.open("w", encoding="utf-8") as f:
            json.dump(out, f, indent=2, ensure_ascii=False)

    # --- queries ---

    def get(self, task_id: str) -> Task | None:
        return self.tasks.get(task_id)

    def all(self) -> list[Task]:
        return list(self.tasks.values())

    def by_status(self, status: Status) -> list[Task]:
        return [t for t in self.tasks.values() if t.status == status]

    def pending_and_in_progress(self) -> list[Task]:
        return [
            t for t in self.tasks.values()
            if t.status in (Status.PENDING, Status.IN_PROGRESS)
        ]

    def by_shortcode(self, shortcode: str) -> list[Task]:
        """Tasks with the given shortcode in pending/in_progress, sorted by file
        ascending then line descending. Same-file fixes go bottom-to-top to avoid
        line-number shifts."""
        ts = [
            t for t in self.pending_and_in_progress()
            if t.issue.shortcode == shortcode
        ]
        ts.sort(key=lambda t: (t.issue.file_path, -t.issue.begin_line))
        return ts

    def counts(self) -> dict[Status, int]:
        out: dict[Status, int] = {s: 0 for s in Status}
        for t in self.tasks.values():
            out[t.status] = out.get(t.status, 0) + 1
        return out

    # --- mutations ---

    def add_if_new(self, issues: Iterable[Issue]) -> int:
        added = 0
        for issue in issues:
            if issue.id in self.tasks:
                continue
            self.tasks[issue.id] = Task.new(issue)
            added += 1
        return added

    def _set_status(self, task_id: str, status: Status, **fields) -> Task:
        t = self.tasks.get(task_id)
        if t is None:
            raise KeyError(f"task not found: {task_id}")
        t.status = status
        t.updated_at = _now()
        for k, v in fields.items():
            setattr(t, k, v)
        return t

    def mark_in_progress(self, ids: list[str]) -> None:
        for i in ids:
            self._set_status(i, Status.IN_PROGRESS)

    def mark_fixed(self, ids: list[str], commit_hash: str, commit_msg: str) -> None:
        now = _now()
        for i in ids:
            self._set_status(
                i, Status.FIXED,
                commit_hash=commit_hash, commit_msg=commit_msg, fixed_at=now,
            )

    def mark_skipped(self, ids: list[str], reason: str = "") -> None:
        for i in ids:
            self._set_status(i, Status.SKIPPED, error_msg=reason)

    def revert_to_pending(self, task_id: str) -> None:
        self._set_status(task_id, Status.PENDING)

    def reset_in_progress(self) -> int:
        in_progress = self.by_status(Status.IN_PROGRESS)
        for t in in_progress:
            t.status = Status.PENDING
            t.updated_at = _now()
        return len(in_progress)

    def reset_all(self) -> None:
        for t in self.tasks.values():
            t.status = Status.PENDING
            t.commit_hash = ""
            t.commit_msg = ""
            t.fixed_at = ""
            t.error_msg = ""
            t.updated_at = _now()
