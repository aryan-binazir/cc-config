#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# ///

from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


QUERY = """\
query(
  $owner: String!,
  $repo: String!,
  $number: Int!,
  $commentsCursor: String,
  $reviewsCursor: String,
  $threadsCursor: String
) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      number
      url
      title
      state
      headRefName

      comments(first: 100, after: $commentsCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          body
          createdAt
          updatedAt
          url
          isMinimized
          minimizedReason
          author { login }
        }
      }

      reviews(first: 100, after: $reviewsCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          state
          body
          submittedAt
          url
          author { login }
        }
      }

      reviewThreads(first: 100, after: $threadsCursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          diffSide
          startLine
          startDiffSide
          originalLine
          originalStartLine
          resolvedBy { login }
          comments(first: 100) {
            nodes {
              id
              body
              createdAt
              updatedAt
              url
              state
              isMinimized
              minimizedReason
              author { login }
            }
          }
        }
      }
    }
  }
}
"""


REPLY_THREAD_MUTATION = """\
mutation($threadId: ID!, $body: String!) {
  addPullRequestReviewThreadReply(input: {pullRequestReviewThreadId: $threadId, body: $body}) {
    comment {
      id
      url
    }
  }
}
"""


RESOLVE_THREAD_MUTATION = """\
mutation($threadId: ID!) {
  resolveReviewThread(input: {threadId: $threadId}) {
    thread {
      id
      isResolved
    }
  }
}
"""


UNRESOLVE_THREAD_MUTATION = """\
mutation($threadId: ID!) {
  unresolveReviewThread(input: {threadId: $threadId}) {
    thread {
      id
      isResolved
    }
  }
}
"""


RESOLUTIONS = {"accepted", "rejected", "deferred"}


class ScriptError(RuntimeError):
    pass


@dataclass(frozen=True)
class PrRef:
    owner: str
    repo: str
    number: int
    title: str
    url: str
    branch: str | None


@dataclass(frozen=True)
class Candidate:
    source_id: str
    type: str
    author: str
    body: str
    created_at: str
    updated_at: str
    url: str | None
    active: bool
    thread_id: str | None = None
    parent_source_id: str | None = None
    path: str | None = None
    line: int | None = None
    start_line: int | None = None
    original_line: int | None = None
    original_start_line: int | None = None
    review_state: str | None = None
    comment_state: str | None = None
    is_resolved: bool | None = None
    is_outdated: bool | None = None
    is_minimized: bool | None = None
    minimized_reason: str | None = None

    @property
    def fingerprint(self) -> str:
        payload = json.dumps(
            {"body": self.body, "updatedAt": self.updated_at},
            sort_keys=True,
            ensure_ascii=False,
        )
        return hashlib.sha256(payload.encode("utf-8")).hexdigest()


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def read_json(path: Path | str) -> dict[str, Any]:
    if str(path) == "-":
        raw = sys.stdin.read()
    else:
        raw = Path(path).read_text(encoding="utf-8")
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as exc:
        raise ScriptError(f"failed to parse JSON: {exc}") from exc
    if not isinstance(data, dict):
        raise ScriptError("input JSON must be an object")
    return data


def write_json(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def run(cmd: list[str], repo: Path | None = None, stdin: str | None = None) -> str:
    result = subprocess.run(
        cmd,
        cwd=str(repo) if repo else None,
        input=stdin,
        text=True,
        capture_output=True,
        check=False,
    )
    if result.returncode != 0:
        detail = (result.stderr or result.stdout).strip()
        raise ScriptError(f"command failed: {' '.join(cmd)}\n{detail}".rstrip())
    return result.stdout


def run_json(cmd: list[str], repo: Path | None = None, stdin: str | None = None) -> dict[str, Any]:
    output = run(cmd, repo=repo, stdin=stdin)
    try:
        data = json.loads(output)
    except json.JSONDecodeError as exc:
        raise ScriptError(f"command did not return JSON: {' '.join(cmd)}") from exc
    if not isinstance(data, dict):
        raise ScriptError(f"command JSON was not an object: {' '.join(cmd)}")
    return data


def current_branch(repo: Path) -> str | None:
    try:
        branch = run(["git", "branch", "--show-current"], repo=repo).strip()
    except ScriptError:
        return None
    return branch or None


def parse_pr_url(url: str) -> tuple[str, str, int]:
    match = re.search(r"github\.com/([^/]+)/([^/]+)/pull/(\d+)", url)
    if not match:
        raise ScriptError(f"could not parse GitHub PR URL: {url}")
    owner, repo, number = match.groups()
    return owner, repo, int(number)


def resolve_current_pr(repo: Path) -> PrRef:
    data = run_json(["gh", "pr", "view", "--json", "number,title,url,headRefName"], repo=repo)
    url = str(data.get("url") or "")
    owner, repo_name, number = parse_pr_url(url)
    return PrRef(
        owner=owner,
        repo=repo_name,
        number=int(data.get("number") or number),
        title=str(data.get("title") or ""),
        url=url,
        branch=str(data.get("headRefName") or current_branch(repo) or ""),
    )


def graphql(repo: Path, ref: PrRef, cursors: dict[str, str | None]) -> dict[str, Any]:
    cmd = [
        "gh",
        "api",
        "graphql",
        "-F",
        "query=@-",
        "-F",
        f"owner={ref.owner}",
        "-F",
        f"repo={ref.repo}",
        "-F",
        f"number={ref.number}",
    ]
    if cursors.get("comments"):
        cmd += ["-F", f"commentsCursor={cursors['comments']}"]
    if cursors.get("reviews"):
        cmd += ["-F", f"reviewsCursor={cursors['reviews']}"]
    if cursors.get("threads"):
        cmd += ["-F", f"threadsCursor={cursors['threads']}"]
    payload = run_json(cmd, repo=repo, stdin=QUERY)
    if payload.get("errors"):
        raise ScriptError("GitHub GraphQL errors:\n" + json.dumps(payload["errors"], indent=2))
    return payload


def graphql_mutation(repo: Path, query: str, fields: dict[str, str]) -> dict[str, Any]:
    cmd = ["gh", "api", "graphql", "-F", "query=@-"]
    for key, value in fields.items():
        cmd += ["-f", f"{key}={value}"]
    payload = run_json(cmd, repo=repo, stdin=query)
    if payload.get("errors"):
        raise ScriptError("GitHub GraphQL errors:\n" + json.dumps(payload["errors"], indent=2))
    return payload


def fetch_pr_data(repo: Path) -> dict[str, Any]:
    run(["gh", "auth", "status"], repo=repo)
    ref = resolve_current_pr(repo)
    comments: list[dict[str, Any]] = []
    reviews: list[dict[str, Any]] = []
    threads: list[dict[str, Any]] = []
    cursors: dict[str, str | None] = {"comments": None, "reviews": None, "threads": None}
    pr_meta: dict[str, Any] | None = None

    while True:
        payload = graphql(repo, ref, cursors)
        pr = payload["data"]["repository"]["pullRequest"]
        if pr_meta is None:
            pr_meta = {
                "number": pr["number"],
                "title": pr["title"],
                "url": pr["url"],
                "state": pr["state"],
                "branch": pr.get("headRefName") or ref.branch,
                "owner": ref.owner,
                "repo": ref.repo,
            }

        comment_page = pr["comments"]
        review_page = pr["reviews"]
        thread_page = pr["reviewThreads"]
        comments.extend(comment_page.get("nodes") or [])
        reviews.extend(review_page.get("nodes") or [])
        threads.extend(thread_page.get("nodes") or [])

        cursors = {
            "comments": comment_page["pageInfo"]["endCursor"] if comment_page["pageInfo"]["hasNextPage"] else None,
            "reviews": review_page["pageInfo"]["endCursor"] if review_page["pageInfo"]["hasNextPage"] else None,
            "threads": thread_page["pageInfo"]["endCursor"] if thread_page["pageInfo"]["hasNextPage"] else None,
        }
        if not any(cursors.values()):
            break

    if pr_meta is None:
        raise ScriptError("failed to fetch PR metadata")
    return {
        "pull_request": pr_meta,
        "conversation_comments": comments,
        "reviews": reviews,
        "review_threads": threads,
    }


def coerce_pr_data(raw: dict[str, Any]) -> dict[str, Any]:
    if "pull_request" in raw:
        return raw

    try:
        pr = raw["data"]["repository"]["pullRequest"]
    except KeyError as exc:
        raise ScriptError("input JSON must contain pull_request or data.repository.pullRequest") from exc

    return {
        "pull_request": {
            "number": pr["number"],
            "title": pr["title"],
            "url": pr["url"],
            "state": pr.get("state"),
            "branch": pr.get("headRefName"),
        },
        "conversation_comments": (pr.get("comments") or {}).get("nodes") or [],
        "reviews": (pr.get("reviews") or {}).get("nodes") or [],
        "review_threads": (pr.get("reviewThreads") or {}).get("nodes") or [],
    }


def author_login(node: dict[str, Any]) -> str:
    author = node.get("author") or {}
    if isinstance(author, dict) and author.get("login"):
        return str(author["login"])
    return "unknown"


def is_comment_active(node: dict[str, Any]) -> bool:
    if node.get("isMinimized") is True:
        return False
    state = node.get("state")
    if state is not None and state not in {"ACTIVE", "SUBMITTED"}:
        return False
    return True


def text(value: Any) -> str:
    return value if isinstance(value, str) else ""


def normalize(raw: dict[str, Any]) -> list[Candidate]:
    data = coerce_pr_data(raw)
    candidates: list[Candidate] = []

    for node in data.get("conversation_comments") or []:
        if not isinstance(node, dict):
            continue
        candidates.append(
            Candidate(
                source_id=str(node["id"]),
                type="issue_comment",
                author=author_login(node),
                body=text(node.get("body")),
                created_at=text(node.get("createdAt")),
                updated_at=text(node.get("updatedAt")) or text(node.get("createdAt")),
                url=text(node.get("url")) or None,
                active=is_comment_active(node),
                is_minimized=node.get("isMinimized"),
                minimized_reason=text(node.get("minimizedReason")) or None,
            )
        )

    for node in data.get("reviews") or []:
        if not isinstance(node, dict):
            continue
        body = text(node.get("body")).strip()
        if not body:
            continue
        submitted_at = text(node.get("submittedAt"))
        candidates.append(
            Candidate(
                source_id=str(node["id"]),
                type="review_summary",
                author=author_login(node),
                body=body,
                created_at=submitted_at,
                updated_at=submitted_at,
                url=text(node.get("url")) or None,
                active=True,
                review_state=text(node.get("state")) or None,
            )
        )

    for thread in data.get("review_threads") or []:
        if not isinstance(thread, dict):
            continue
        comments = ((thread.get("comments") or {}).get("nodes") or [])
        active_comments = [comment for comment in comments if isinstance(comment, dict) and is_comment_active(comment)]
        first_active_id = str(active_comments[0]["id"]) if active_comments else None
        thread_active = not bool(thread.get("isResolved")) and not bool(thread.get("isOutdated"))

        for index, node in enumerate(active_comments):
            source_id = str(node["id"])
            active = thread_active and is_comment_active(node)
            parent_source_id = None if index == 0 else first_active_id
            candidates.append(
                Candidate(
                    source_id=source_id,
                    type="review_comment",
                    author=author_login(node),
                    body=text(node.get("body")),
                    created_at=text(node.get("createdAt")),
                    updated_at=text(node.get("updatedAt")) or text(node.get("createdAt")),
                    url=text(node.get("url")) or None,
                    active=active,
                    thread_id=str(thread["id"]),
                    parent_source_id=parent_source_id,
                    path=text(thread.get("path")) or None,
                    line=thread.get("line"),
                    start_line=thread.get("startLine"),
                    original_line=thread.get("originalLine"),
                    original_start_line=thread.get("originalStartLine"),
                    comment_state=text(node.get("state")) or None,
                    is_resolved=thread.get("isResolved"),
                    is_outdated=thread.get("isOutdated"),
                    is_minimized=node.get("isMinimized"),
                    minimized_reason=text(node.get("minimizedReason")) or None,
                )
            )

    return candidates


def state_path(repo: Path, state_dir: Path | None, pr_number: int) -> Path:
    root = state_dir if state_dir else repo / "_scratch" / "_pr_reviews"
    return root / f"pr-{pr_number}.json"


def initial_state(raw: dict[str, Any], repo: Path) -> dict[str, Any]:
    pr = coerce_pr_data(raw)["pull_request"]
    return {
        "pr": {
            "number": int(pr["number"]),
            "title": text(pr.get("title")),
            "url": text(pr.get("url")),
            "branch": text(pr.get("branch")) or current_branch(repo),
        },
        "nextNumber": 1,
        "itemsById": {},
    }


def load_state(path: Path, raw: dict[str, Any] | None, repo: Path) -> dict[str, Any]:
    if path.exists():
        state = read_json(path)
    elif raw is not None:
        state = initial_state(raw, repo)
    else:
        raise ScriptError(f"state file does not exist: {path}")
    state.setdefault("nextNumber", 1)
    state.setdefault("itemsById", {})
    return state


def item_fields(candidate: Candidate, number: str, now: str) -> dict[str, Any]:
    return {
        "number": number,
        "type": candidate.type,
        "threadId": candidate.thread_id,
        "parentSourceId": candidate.parent_source_id,
        "createdAt": candidate.created_at,
        "updatedAt": candidate.updated_at,
        "author": candidate.author,
        "body": candidate.body,
        "active": candidate.active,
        "resolved": candidate.is_resolved,
        "outdated": candidate.is_outdated,
        "minimized": candidate.is_minimized,
        "minimizedReason": candidate.minimized_reason,
        "commentState": candidate.comment_state,
        "reviewState": candidate.review_state,
        "path": candidate.path,
        "line": candidate.line,
        "startLine": candidate.start_line,
        "originalLine": candidate.original_line,
        "originalStartLine": candidate.original_start_line,
        "url": candidate.url,
        "lastSeenAt": now,
        "lastSeenFingerprint": candidate.fingerprint,
    }


def assign_top_number(state: dict[str, Any]) -> str:
    number = int(state.get("nextNumber") or 1)
    state["nextNumber"] = number + 1
    return str(number)


def assign_reply_number(items: dict[str, Any], thread_id: str | None, parent_number: str) -> str:
    highest = 0
    prefix = parent_number + "."
    for item in items.values():
        if not isinstance(item, dict):
            continue
        if item.get("threadId") != thread_id:
            continue
        number = str(item.get("number") or "")
        if not number.startswith(prefix):
            continue
        suffix = number[len(prefix) :]
        if suffix.isdigit():
            highest = max(highest, int(suffix))
    return f"{parent_number}.{highest + 1}"


def clear_resolution(item: dict[str, Any]) -> None:
    for key in ("resolution", "resolutionNote", "resolutionAt"):
        item.pop(key, None)


def upsert_candidate(state: dict[str, Any], candidate: Candidate, number: str, now: str) -> None:
    items = state["itemsById"]
    existing = items.get(candidate.source_id)
    changed = False
    became_active = False
    if isinstance(existing, dict):
        old_fingerprint = existing.get("lastSeenFingerprint")
        if old_fingerprint:
            changed = old_fingerprint != candidate.fingerprint
        else:
            changed = existing.get("body") != candidate.body or existing.get("updatedAt") != candidate.updated_at
        became_active = existing.get("active") is False and candidate.active
        status = existing.get("status") or "open"
        merged = dict(existing)
    else:
        status = "open"
        merged = {}

    merged.update(item_fields(candidate, number, now))
    if candidate.active and (changed or became_active):
        status = "open"
        clear_resolution(merged)
    merged["status"] = status
    items[candidate.source_id] = merged


def mark_missing_inactive(state: dict[str, Any], seen_ids: set[str]) -> None:
    for source_id, item in state["itemsById"].items():
        if not isinstance(item, dict):
            continue
        if source_id not in seen_ids:
            item["active"] = False


def merge(raw: dict[str, Any], state: dict[str, Any], repo: Path) -> dict[str, Any]:
    now = utc_now()
    pr = coerce_pr_data(raw)["pull_request"]
    state["pr"] = {
        "number": int(pr["number"]),
        "title": text(pr.get("title")),
        "url": text(pr.get("url")),
        "branch": text(pr.get("branch")) or current_branch(repo),
    }
    state.setdefault("nextNumber", 1)
    items = state.setdefault("itemsById", {})
    candidates = normalize(raw)
    seen_ids = {candidate.source_id for candidate in candidates}

    inactive_existing = [candidate for candidate in candidates if not candidate.active and candidate.source_id in items]
    for candidate in inactive_existing:
        existing_number = str(items[candidate.source_id].get("number") or "")
        if existing_number:
            upsert_candidate(state, candidate, existing_number, now)

    active_tops = sorted(
        [candidate for candidate in candidates if candidate.active and candidate.parent_source_id is None],
        key=lambda candidate: (candidate.created_at, candidate.source_id),
    )
    for candidate in active_tops:
        existing = items.get(candidate.source_id)
        number = str(existing.get("number")) if isinstance(existing, dict) and existing.get("number") else assign_top_number(state)
        upsert_candidate(state, candidate, number, now)

    active_replies = sorted(
        [candidate for candidate in candidates if candidate.active and candidate.parent_source_id is not None],
        key=lambda candidate: (candidate.created_at, candidate.source_id),
    )
    for candidate in active_replies:
        existing = items.get(candidate.source_id)
        if isinstance(existing, dict) and existing.get("number"):
            number = str(existing["number"])
        else:
            parent = items.get(candidate.parent_source_id or "")
            if not isinstance(parent, dict) or not parent.get("number"):
                number = assign_top_number(state)
            else:
                number = assign_reply_number(items, candidate.thread_id, str(parent["number"]))
        upsert_candidate(state, candidate, number, now)

    mark_missing_inactive(state, seen_ids)
    return state


def number_key(number: str) -> tuple[int, int]:
    parts = number.split(".", 1)
    try:
        top = int(parts[0])
    except ValueError:
        top = 10**9
    if len(parts) == 1:
        return top, 0
    try:
        reply = int(parts[1])
    except ValueError:
        reply = 10**9
    return top, reply


def excerpt(body: str, limit: int = 180) -> str:
    compact = re.sub(r"\s+", " ", body).strip()
    if len(compact) <= limit:
        return compact
    return compact[: limit - 3].rstrip() + "..."


def location(item: dict[str, Any]) -> str:
    path = item.get("path")
    line = item.get("line") or item.get("originalLine")
    if path and line:
        return f" at {path}:{line}"
    if path:
        return f" at {path}"
    return ""


def render_item(item: dict[str, Any]) -> str:
    status = item.get("status") or "open"
    text_value = (
        f"{item.get('type', 'comment')} by {item.get('author', 'unknown')}"
        f"{location(item)} - {excerpt(text(item.get('body')))}"
    )
    if status == "handled":
        text_value = f"~~{text_value}~~"
    return f"{item.get('number')} [{status}] {text_value}"


def render(state: dict[str, Any]) -> str:
    pr = state.get("pr") or {}
    title = text(pr.get("title")) or f"PR #{pr.get('number', '')}".strip()
    url = text(pr.get("url"))
    lines = [f"PR: [{title}]({url})" if url else f"PR: {title}", ""]

    active_items = [
        item
        for item in (state.get("itemsById") or {}).values()
        if isinstance(item, dict) and item.get("active") is True and item.get("number")
    ]
    active_items.sort(key=lambda item: number_key(str(item["number"])))

    if not active_items:
        lines.append("(No active PR comments.)")
    else:
        for item in active_items:
            number = str(item.get("number") or "")
            prefix = "  " if "." in number else ""
            lines.append(prefix + render_item(item))

    lines.extend(["", "Pick a number to discuss."])
    return "\n".join(lines)


def sync(args: argparse.Namespace) -> int:
    repo = args.repo.resolve()
    raw = coerce_pr_data(read_json(args.input_json)) if args.input_json else fetch_pr_data(repo)
    pr_number = int(raw["pull_request"]["number"])
    path = state_path(repo, args.state_dir, pr_number)
    state = load_state(path, raw, repo)
    state = merge(raw, state, repo)
    if not args.dry_run:
        write_json(path, state)
    print(render(state))
    return 0


def show(args: argparse.Namespace) -> int:
    repo = args.repo.resolve()
    pr_number = args.pr or resolve_current_pr(repo).number
    path = state_path(repo, args.state_dir, pr_number)
    state = load_state(path, None, repo)
    print(render(state))
    return 0


def load_state_for_pr(args: argparse.Namespace) -> tuple[Path, dict[str, Any], Path]:
    repo = args.repo.resolve()
    pr_number = args.pr or resolve_current_pr(repo).number
    path = state_path(repo, args.state_dir, pr_number)
    return repo, load_state(path, None, repo), path


def find_item_by_number(state: dict[str, Any], number: str) -> dict[str, Any]:
    for item in state["itemsById"].values():
        if isinstance(item, dict) and str(item.get("number")) == number:
            return item
    pr_number = (state.get("pr") or {}).get("number", "unknown")
    raise ScriptError(f"could not find PR comment number {number} in PR {pr_number}")


def items_in_thread(state: dict[str, Any], thread_id: str) -> list[dict[str, Any]]:
    return [
        item
        for item in state["itemsById"].values()
        if isinstance(item, dict) and item.get("threadId") == thread_id
    ]


def mark_item_handled(item: dict[str, Any], resolution: str, note: str) -> None:
    item["status"] = "handled"
    item["resolution"] = resolution
    item["resolutionNote"] = note
    item["resolutionAt"] = utc_now()


def body_from_args(body: str | None, body_file: Path | None, label: str) -> str:
    if body_file:
        value = body_file.read_text(encoding="utf-8")
    else:
        value = body or ""
    value = value.strip()
    if not value:
        raise ScriptError(f"{label} cannot be empty")
    return value


def mark(args: argparse.Namespace) -> int:
    if args.resolution not in RESOLUTIONS:
        raise ScriptError(f"resolution must be one of: {', '.join(sorted(RESOLUTIONS))}")
    _, state, path = load_state_for_pr(args)
    target = find_item_by_number(state, args.number)
    mark_item_handled(target, args.resolution, args.note)
    write_json(path, state)
    print(render(state))
    return 0


def post_reply(repo: Path, state: dict[str, Any], item: dict[str, Any], body: str) -> dict[str, Any]:
    thread_id = text(item.get("threadId"))
    if thread_id:
        payload = graphql_mutation(repo, REPLY_THREAD_MUTATION, {"threadId": thread_id, "body": body})
        comment = payload["data"]["addPullRequestReviewThreadReply"]["comment"]
        return {"kind": "review_thread_reply", "id": comment.get("id"), "url": comment.get("url")}

    pr_number = str((state.get("pr") or {}).get("number") or "")
    if not pr_number:
        raise ScriptError("state is missing pr.number")
    output = run(["gh", "pr", "comment", pr_number, "--body", body], repo=repo).strip()
    return {"kind": "pr_comment", "url": output or None}


def set_thread_resolution(repo: Path, state: dict[str, Any], item: dict[str, Any], resolved: bool) -> dict[str, Any]:
    thread_id = text(item.get("threadId"))
    if not thread_id:
        raise ScriptError(f"comment number {item.get('number')} is not a review thread comment")
    mutation = RESOLVE_THREAD_MUTATION if resolved else UNRESOLVE_THREAD_MUTATION
    payload = graphql_mutation(repo, mutation, {"threadId": thread_id})
    key = "resolveReviewThread" if resolved else "unresolveReviewThread"
    thread = payload["data"][key]["thread"]

    now = utc_now()
    for thread_item in items_in_thread(state, thread_id):
        thread_item["resolved"] = resolved
        thread_item["active"] = not resolved
        if resolved:
            thread_item["githubResolvedAt"] = now
        else:
            thread_item["githubUnresolvedAt"] = now
            thread_item["active"] = True
    return {"threadId": thread.get("id"), "isResolved": thread.get("isResolved")}


def reply(args: argparse.Namespace) -> int:
    repo, state, path = load_state_for_pr(args)
    target = find_item_by_number(state, args.number)
    body = body_from_args(args.body, args.body_file, "reply body")
    result = post_reply(repo, state, target, body)
    target.setdefault("githubReplies", []).append({"body": body, "createdAt": utc_now(), **result})
    write_json(path, state)
    print(render(state))
    if result.get("url"):
        print(f"\nPosted reply: {result['url']}")
    return 0


def resolve(args: argparse.Namespace) -> int:
    repo, state, path = load_state_for_pr(args)
    target = find_item_by_number(state, args.number)
    result = set_thread_resolution(repo, state, target, resolved=True)
    write_json(path, state)
    print(render(state))
    print(f"\nResolved GitHub review thread: {result['threadId']}")
    return 0


def unresolve(args: argparse.Namespace) -> int:
    repo, state, path = load_state_for_pr(args)
    target = find_item_by_number(state, args.number)
    result = set_thread_resolution(repo, state, target, resolved=False)
    write_json(path, state)
    print(render(state))
    print(f"\nReopened GitHub review thread: {result['threadId']}")
    return 0


def accept(args: argparse.Namespace) -> int:
    repo, state, path = load_state_for_pr(args)
    target = find_item_by_number(state, args.number)
    mark_item_handled(target, "accepted", args.note)

    reply_result = None
    reply_body = body_from_args(args.reply, args.reply_file, "reply body") if args.reply or args.reply_file else None
    if reply_body:
        reply_result = post_reply(repo, state, target, reply_body)
        target.setdefault("githubReplies", []).append({"body": reply_body, "createdAt": utc_now(), **reply_result})

    resolve_result = None
    if args.resolve:
        resolve_result = set_thread_resolution(repo, state, target, resolved=True)

    write_json(path, state)
    print(render(state))
    if reply_result and reply_result.get("url"):
        print(f"\nPosted reply: {reply_result['url']}")
    if resolve_result:
        print(f"Resolved GitHub review thread: {resolve_result['threadId']}")
    return 0


def add_common(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--repo", type=Path, default=Path.cwd(), help="Repository root. Defaults to the current directory.")
    parser.add_argument(
        "--state-dir",
        type=Path,
        help="Override the review state directory. Defaults to <repo>/_scratch/_pr_reviews.",
    )


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description=(
            "Fetch, merge, render, mark, reply to, and resolve stable PR comment checklists. "
            "GitHub write commands are explicit: reply, resolve, unresolve, and accept with --reply/--resolve."
        )
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    sync_parser = subparsers.add_parser("sync", help="Fetch PR comments, merge state, persist it, and render the checklist.")
    add_common(sync_parser)
    sync_parser.add_argument("--input-json", help="Read pre-fetched PR JSON from a file, or '-' for stdin.")
    sync_parser.add_argument("--dry-run", action="store_true", help="Render merged state without writing it.")
    sync_parser.set_defaults(func=sync)

    show_parser = subparsers.add_parser("show", help="Render the saved checklist without fetching GitHub.")
    add_common(show_parser)
    show_parser.add_argument("--pr", type=int, help="PR number. Defaults to the current branch PR.")
    show_parser.set_defaults(func=show)

    mark_parser = subparsers.add_parser("mark", help="Mark a checklist item handled locally after discussion.")
    add_common(mark_parser)
    mark_parser.add_argument("--pr", type=int, help="PR number. Defaults to the current branch PR.")
    mark_parser.add_argument("--number", required=True, help="Stable checklist number, for example 3 or 3.1.")
    mark_parser.add_argument("--resolution", required=True, choices=sorted(RESOLUTIONS))
    mark_parser.add_argument("--note", required=True, help="Short resolution note.")
    mark_parser.set_defaults(func=mark)

    accept_parser = subparsers.add_parser(
        "accept",
        help="Mark a numbered item accepted locally; optionally reply on GitHub and resolve its review thread.",
        description=(
            "Mark a numbered item accepted in local triage state. Add --reply/--reply-file to post a GitHub reply. "
            "Add --resolve to resolve the GitHub review thread when the item is an inline review comment."
        ),
    )
    add_common(accept_parser)
    accept_parser.add_argument("--pr", type=int, help="PR number. Defaults to the current branch PR.")
    accept_parser.add_argument("--number", required=True, help="Stable checklist number, for example 3 or 3.1.")
    accept_parser.add_argument("--note", required=True, help="Short local acceptance note.")
    accept_reply = accept_parser.add_mutually_exclusive_group()
    accept_reply.add_argument("--reply", help="GitHub reply body to post while accepting.")
    accept_reply.add_argument("--reply-file", type=Path, help="File containing the GitHub reply body to post while accepting.")
    accept_parser.add_argument("--resolve", action="store_true", help="Also resolve the GitHub review thread.")
    accept_parser.set_defaults(func=accept)

    reply_parser = subparsers.add_parser(
        "reply",
        help="Post a GitHub reply for a numbered item.",
        description=(
            "Post a GitHub reply for a stable checklist number. Inline review comments get a review-thread reply. "
            "Issue comments and review summaries get a top-level PR conversation comment."
        ),
    )
    add_common(reply_parser)
    reply_parser.add_argument("--pr", type=int, help="PR number. Defaults to the current branch PR.")
    reply_parser.add_argument("--number", required=True, help="Stable checklist number, for example 3 or 3.1.")
    reply_body = reply_parser.add_mutually_exclusive_group(required=True)
    reply_body.add_argument("--body", help="GitHub reply body.")
    reply_body.add_argument("--body-file", type=Path, help="File containing the GitHub reply body.")
    reply_parser.set_defaults(func=reply)

    resolve_parser = subparsers.add_parser(
        "resolve",
        help="Resolve the GitHub review thread for a numbered inline review comment.",
        description="Resolve the GitHub review thread for a stable checklist number that points at an inline review comment.",
    )
    add_common(resolve_parser)
    resolve_parser.add_argument("--pr", type=int, help="PR number. Defaults to the current branch PR.")
    resolve_parser.add_argument("--number", required=True, help="Stable checklist number, for example 3 or 3.1.")
    resolve_parser.set_defaults(func=resolve)

    unresolve_parser = subparsers.add_parser(
        "unresolve",
        help="Reopen the GitHub review thread for a numbered inline review comment.",
        description="Reopen the GitHub review thread for a stable checklist number that points at an inline review comment.",
    )
    add_common(unresolve_parser)
    unresolve_parser.add_argument("--pr", type=int, help="PR number. Defaults to the current branch PR.")
    unresolve_parser.add_argument("--number", required=True, help="Stable checklist number, for example 3 or 3.1.")
    unresolve_parser.set_defaults(func=unresolve)

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    try:
        return args.func(args)
    except ScriptError as exc:
        print(str(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
