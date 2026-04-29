"""DeepSource GraphQL API client + glob-based path filter.

Uses urllib (stdlib) to avoid extra deps. Handles two layers of pagination:
issues, then occurrences within each issue.
"""

from __future__ import annotations

import fnmatch
import json
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from typing import Iterator

from store import Issue

API_ENDPOINT = "https://api.deepsource.io/graphql/"


@dataclass
class IssueFilter:
    categories: list[str] = field(default_factory=list)
    severities: list[str] = field(default_factory=list)
    limit: int = 0
    paths_include: list[str] = field(default_factory=list)
    paths_exclude: list[str] = field(default_factory=list)


# --- GraphQL queries ---

REPO_ISSUES_QUERY = """
query GetRepositoryIssues($owner: String!, $name: String!, $provider: VCSProvider!, $first: Int, $after: String) {
  repository(login: $owner, name: $name, vcsProvider: $provider) {
    issues(first: $first, after: $after) {
      edges {
        node {
          id
          issue { shortcode title category severity description analyzer { shortcode } }
          occurrences(first: 100) {
            edges { node { id path beginLine endLine } }
            pageInfo { hasNextPage endCursor }
          }
        }
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}
"""

OCCURRENCES_QUERY = """
query GetOccurrences($issueId: ID!, $first: Int, $after: String) {
  node(id: $issueId) {
    ... on RepositoryIssue {
      occurrences(first: $first, after: $after) {
        edges { node { id path beginLine endLine } }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}
"""


class APIError(RuntimeError):
    pass


class Client:
    def __init__(self, api_token: str, endpoint: str = API_ENDPOINT):
        self.api_token = api_token
        self.endpoint = endpoint

    def _request(self, query: str, variables: dict) -> dict:
        body = json.dumps({"query": query, "variables": variables}).encode("utf-8")
        req = urllib.request.Request(
            self.endpoint,
            data=body,
            method="POST",
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {self.api_token}",
            },
        )
        try:
            with urllib.request.urlopen(req, timeout=30) as resp:
                payload = json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            detail = e.read().decode("utf-8", errors="replace")
            raise APIError(f"API returned status {e.code}: {detail}") from e
        except urllib.error.URLError as e:
            raise APIError(f"network error: {e.reason}") from e

        if payload.get("errors"):
            msg = payload["errors"][0].get("message", "unknown GraphQL error")
            raise APIError(f"GraphQL error: {msg}")
        return payload.get("data") or {}

    # --- public methods ---

    def fetch_issues(self, owner: str, repo: str, flt: IssueFilter | None = None) -> list[Issue]:
        flt = flt or IssueFilter()
        max_issues = flt.limit if flt.limit > 0 else 0
        out: list[Issue] = []
        cursor: str | None = None

        while True:
            variables: dict = {
                "owner": owner,
                "name": repo,
                "provider": "GITHUB",
                "first": 50,
            }
            if cursor:
                variables["after"] = cursor

            data = self._request(REPO_ISSUES_QUERY, variables)
            issues_page = (data.get("repository") or {}).get("issues") or {}

            for edge in issues_page.get("edges", []):
                node = edge["node"]
                issue_meta = node.get("issue") or {}
                analyzer = (issue_meta.get("analyzer") or {}).get("shortcode", "")

                # Collect occurrences (first page from inline query, paginate the rest)
                occ_page = node.get("occurrences") or {}
                occurrences = list(self._iter_occurrences_inline(occ_page))
                if (occ_page.get("pageInfo") or {}).get("hasNextPage"):
                    occurrences = list(self._fetch_all_occurrences(node["id"]))

                for occ in occurrences:
                    issue = Issue(
                        id=occ["id"],
                        title=issue_meta.get("title", ""),
                        category=issue_meta.get("category", ""),
                        shortcode=issue_meta.get("shortcode", ""),
                        severity=issue_meta.get("severity", ""),
                        file_path=occ["path"],
                        begin_line=int(occ.get("beginLine") or 0),
                        end_line=int(occ.get("endLine") or 0),
                        description=issue_meta.get("description", ""),
                        analyzer=analyzer,
                    )

                    if flt.categories and issue.category not in flt.categories:
                        continue
                    if flt.severities and issue.severity not in flt.severities:
                        continue
                    if not path_allowed(issue.file_path, flt.paths_include, flt.paths_exclude):
                        continue

                    out.append(issue)
                    if max_issues and len(out) >= max_issues:
                        return out[:max_issues]

            page_info = issues_page.get("pageInfo") or {}
            if not page_info.get("hasNextPage"):
                break
            cursor = page_info.get("endCursor")

        return out

    def _iter_occurrences_inline(self, occ_page: dict) -> Iterator[dict]:
        for edge in occ_page.get("edges", []):
            yield edge["node"]

    def _fetch_all_occurrences(self, issue_id: str) -> Iterator[dict]:
        cursor: str | None = None
        while True:
            variables: dict = {"issueId": issue_id, "first": 100}
            if cursor:
                variables["after"] = cursor
            data = self._request(OCCURRENCES_QUERY, variables)
            occ_page = ((data.get("node") or {}).get("occurrences")) or {}
            for edge in occ_page.get("edges", []):
                yield edge["node"]
            page_info = occ_page.get("pageInfo") or {}
            if not page_info.get("hasNextPage"):
                return
            cursor = page_info.get("endCursor")


# --- glob path filter ---

def path_allowed(path: str, include: list[str], exclude: list[str]) -> bool:
    """Return True if path passes both include and exclude filters.

    Empty include allows everything by default. Patterns support ``**`` to match
    across directory separators (otherwise behaves like fnmatch per segment).
    """
    if include and not any(_match_path(p, path) for p in include):
        return False
    return not any(_match_path(p, path) for p in exclude)


def _match_path(pattern: str, path: str) -> bool:
    return _match_segments(pattern.split("/"), path.split("/"))


def _match_segments(pat: list[str], parts: list[str]) -> bool:
    if not pat:
        return not parts
    if pat[0] == "**":
        for j in range(len(parts) + 1):
            if _match_segments(pat[1:], parts[j:]):
                return True
        return False
    if not parts:
        return False
    if not fnmatch.fnmatchcase(parts[0], pat[0]):
        return False
    return _match_segments(pat[1:], parts[1:])
