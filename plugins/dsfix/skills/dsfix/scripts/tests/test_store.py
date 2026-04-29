"""Tests for task store: add/save/load roundtrip + status transitions + sort."""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))

from store import Issue, Status, Store  # noqa: E402


def _issue(id_: str, shortcode: str = "SC1", path: str = "a.go", line: int = 1) -> Issue:
    return Issue(id=id_, shortcode=shortcode, file_path=path, begin_line=line, end_line=line, title="t")


def test_add_save_load_roundtrip(tmp_path: Path):
    s1 = Store(tmp_path)
    added = s1.add_if_new([_issue("a"), _issue("b")])
    s1.save()
    assert added == 2

    s2 = Store(tmp_path)
    assert {t.id for t in s2.all()} == {"a", "b"}


def test_add_if_new_dedups(tmp_path: Path):
    s = Store(tmp_path)
    assert s.add_if_new([_issue("a"), _issue("a")]) == 1
    assert s.add_if_new([_issue("a"), _issue("b")]) == 1


def test_status_transitions(tmp_path: Path):
    s = Store(tmp_path)
    s.add_if_new([_issue("a")])
    s.mark_in_progress(["a"])
    assert s.by_status(Status.IN_PROGRESS)[0].id == "a"
    s.mark_fixed(["a"], "abc1234", "fix(SC1): t")
    t = s.get("a")
    assert t is not None
    assert t.status == Status.FIXED
    assert t.commit_hash == "abc1234"


def test_revert_to_pending(tmp_path: Path):
    s = Store(tmp_path)
    s.add_if_new([_issue("a")])
    s.mark_in_progress(["a"])
    s.revert_to_pending("a")
    assert s.get("a").status == Status.PENDING


def test_reset_in_progress(tmp_path: Path):
    s = Store(tmp_path)
    s.add_if_new([_issue("a"), _issue("b")])
    s.mark_in_progress(["a", "b"])
    n = s.reset_in_progress()
    assert n == 2
    assert all(t.status == Status.PENDING for t in s.all())


def test_by_shortcode_sort_order(tmp_path: Path):
    """Same file should be returned line-desc; different files in file-asc."""
    s = Store(tmp_path)
    s.add_if_new([
        _issue("1", path="a.go", line=10),
        _issue("2", path="a.go", line=20),
        _issue("3", path="b.go", line=5),
    ])
    ts = s.by_shortcode("SC1")
    assert [t.id for t in ts] == ["2", "1", "3"]  # a.go:20, a.go:10, b.go:5
