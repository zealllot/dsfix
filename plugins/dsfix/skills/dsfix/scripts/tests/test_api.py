"""Tests for the glob-aware path filter in api.path_allowed."""

from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))

from api import path_allowed, _match_path  # noqa: E402


def test_match_path_basic():
    assert _match_path("internal/**", "internal/foo.go")
    assert _match_path("internal/**", "internal/foo/bar.go")
    assert not _match_path("internal/**", "cmd/foo.go")

    assert _match_path("**/*_test.go", "internal/foo_test.go")
    assert _match_path("**/*_test.go", "deeply/nested/dir/x_test.go")
    assert not _match_path("**/*_test.go", "internal/foo.go")

    assert _match_path("vendor/**", "vendor/x/y/z.go")
    assert not _match_path("vendor/**", "internal/vendor/x.go")

    assert _match_path("*.go", "foo.go")
    assert not _match_path("*.go", "sub/foo.go")

    assert _match_path("a/b/c.go", "a/b/c.go")
    assert not _match_path("a/b/c.go", "a/b/d.go")

    assert _match_path("**", "anything/at/all.go")


def test_path_allowed_no_filters():
    assert path_allowed("any/path.go", [], [])


def test_path_allowed_include_must_match():
    assert path_allowed("internal/x.go", ["internal/**"], [])
    assert not path_allowed("cmd/x.go", ["internal/**"], [])


def test_path_allowed_exclude_wins():
    assert not path_allowed(
        "internal/x_test.go",
        ["internal/**"],
        ["**/*_test.go"],
    )


def test_path_allowed_exclude_only():
    assert path_allowed("internal/x.go", [], ["vendor/**"])
    assert not path_allowed("vendor/x/y.go", [], ["vendor/**"])
