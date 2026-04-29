"""Configuration loading for dsfix.

Reads .dsfix.yaml at repo root. Falls back to environment variable for the API
token. Auto-installs PyYAML on first use if missing.
"""

from __future__ import annotations

import os
import sys
from dataclasses import dataclass, field
from pathlib import Path


try:
    import yaml
except ImportError:
    msg = (
        "dsfix: PyYAML is required but not installed.\n"
        "Install it with one of:\n"
        f"  {sys.executable} -m pip install --user PyYAML\n"
        f"  pipx install PyYAML\n"
        f"  {sys.executable} -m pip install --break-system-packages PyYAML  (PEP 668 systems)\n"
    )
    sys.stderr.write(msg)
    sys.exit(1)

CONFIG_FILENAME = ".dsfix.yaml"


@dataclass
class FilterConfig:
    categories: list[str] = field(default_factory=list)
    severities: list[str] = field(default_factory=list)
    limit: int = 0
    paths_include: list[str] = field(default_factory=list)
    paths_exclude: list[str] = field(default_factory=list)


@dataclass
class Config:
    api_token: str = ""
    owner: str = ""
    name: str = ""
    filter: FilterConfig = field(default_factory=FilterConfig)
    verify_command: str = ""

    def validate(self) -> None:
        if not self.api_token:
            raise ValueError(
                "DeepSource API token is required (set in config or DEEPSOURCE_API_TOKEN env var)"
            )
        if not self.owner:
            raise ValueError("repository owner is required")
        if not self.name:
            raise ValueError("repository name is required")


def load(repo_path: str | Path) -> Config:
    """Load config from <repo_path>/.dsfix.yaml. Env var overrides yaml token."""
    path = Path(repo_path) / CONFIG_FILENAME
    if not path.exists():
        raise FileNotFoundError(f"config not found: {path} (run `dsfix init` to create one)")

    with path.open("r", encoding="utf-8") as f:
        data = yaml.safe_load(f) or {}

    deepsource = data.get("deepsource") or {}
    repo = data.get("repository") or {}
    flt = data.get("filter") or {}
    verify = data.get("verify") or {}

    cfg = Config(
        api_token=str(deepsource.get("api_token") or ""),
        owner=str(repo.get("owner") or ""),
        name=str(repo.get("name") or ""),
        filter=FilterConfig(
            categories=list(flt.get("categories") or []),
            severities=list(flt.get("severities") or []),
            limit=int(flt.get("limit") or 0),
            paths_include=list(flt.get("paths_include") or []),
            paths_exclude=list(flt.get("paths_exclude") or []),
        ),
        verify_command=str(verify.get("command") or ""),
    )

    env_token = os.environ.get("DEEPSOURCE_API_TOKEN", "")
    if cfg.api_token and not env_token:
        print(
            f"warning: api_token in {path} is a secret — prefer DEEPSOURCE_API_TOKEN env var",
            file=sys.stderr,
        )
    if env_token:
        cfg.api_token = env_token

    return cfg


TEMPLATE = """# DSFix Configuration

deepsource:
  # Prefer setting DEEPSOURCE_API_TOKEN env var instead of putting the token here.
  api_token: ""

repository:
  # VCS owner/organization
  owner: ""
  # Repository name
  name: ""

filter:
  # Categories to include (leave empty for all)
  # Options: Bug Risk, Anti-pattern, Security, Performance, Typecheck, Style, Documentation
  categories: []

  # Severities to include (leave empty for all)
  # Options: critical, major, minor
  severities: []

  # Maximum number of issues to fetch (leave empty for unlimited)
  limit:

  # Optional path glob filters (supports ** for cross-directory matching).
  # Examples:
  #   paths_include: ["internal/**", "cmd/**"]
  #   paths_exclude: ["vendor/**", "**/*_test.go"]
  paths_include: []
  paths_exclude: []

verify:
  # Command run after a fix to verify it compiles/passes. Leave empty to let the AI pick.
  # Examples: "go build ./...", "pnpm typecheck", "make lint"
  command: ""
"""


def write_template(repo_path: str | Path) -> Path:
    """Write the config template to <repo_path>/.dsfix.yaml. Errors if it exists."""
    path = Path(repo_path) / CONFIG_FILENAME
    if path.exists():
        raise FileExistsError(f"config already exists: {path}")
    path.write_text(TEMPLATE, encoding="utf-8")
    return path
