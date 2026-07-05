#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# ///

from __future__ import annotations

import argparse
import json
import re
import shlex
import shutil
import subprocess
import time
from pathlib import Path
from typing import Any


RUNNER_BINARIES = {
    "claude": "claude",
    "codex": "codex",
    "cursor": "cursor-agent",
    "cursor-agent": "cursor-agent",
}


def emit(payload: dict[str, Any], pretty: bool = False) -> None:
    print(json.dumps(payload, indent=2 if pretty else None, sort_keys=pretty))


def run_cmd(cmd: list[str], cwd: Path | None = None, timeout_ms: int = 10_000) -> dict[str, Any]:
    started = time.monotonic()
    try:
        result = subprocess.run(
            cmd,
            cwd=str(cwd) if cwd else None,
            text=True,
            capture_output=True,
            timeout=timeout_ms / 1000,
            check=False,
        )
        return {
            "ok": result.returncode == 0,
            "exit_code": result.returncode,
            "timed_out": False,
            "elapsed_ms": int((time.monotonic() - started) * 1000),
            "stdout": result.stdout,
            "stderr": result.stderr,
        }
    except subprocess.TimeoutExpired as exc:
        return {
            "ok": False,
            "exit_code": None,
            "timed_out": True,
            "elapsed_ms": int((time.monotonic() - started) * 1000),
            "stdout": exc.stdout or "",
            "stderr": exc.stderr or "",
        }


def git_output(repo: Path, args: list[str], timeout_ms: int = 10_000) -> dict[str, Any]:
    return run_cmd(["git", *args], cwd=repo, timeout_ms=timeout_ms)


def current_branch(repo: Path) -> str | None:
    result = git_output(repo, ["branch", "--show-current"])
    branch = result["stdout"].strip()
    return branch or None


def compact_lines(value: str, limit: int = 40) -> list[str]:
    return [line for line in value.splitlines() if line.strip()][:limit]


def detect_ticket_key(*values: str | None) -> str | None:
    for value in values:
        if not value:
            continue
        match = re.search(r"\b[A-Z][A-Z0-9]+-\d+\b", value)
        if match:
            return match.group(0)
    return None


def rule_files(repo: Path) -> list[str]:
    candidates = [repo / "AGENTS.md", repo / "CLAUDE.md", repo / ".cursorrules"]
    return [str(path) for path in candidates if path.exists()]


def split_csv(value: str | None) -> list[str]:
    if not value:
        return []
    return [item.strip() for item in value.split(",") if item.strip()]


def runner_available(runner: str) -> dict[str, Any]:
    binary = RUNNER_BINARIES.get(runner, runner)
    path = shutil.which(binary)
    return {"runner": runner, "binary": binary, "available": path is not None, "path": path}


def command_string(args: list[str | Path]) -> str:
    return " ".join(shlex.quote(str(arg)) for arg in args)


def repo_facts(
    repo: Path,
    input_value: str,
    critic_runner: str | None,
    review_runners: str | None,
    headless_runner: str | None,
    timeout_ms: int,
) -> dict[str, Any]:
    repo = repo.resolve()
    blockers: list[str] = []

    worktree = git_output(repo, ["rev-parse", "--is-inside-work-tree"])
    is_worktree = worktree["ok"] and worktree["stdout"].strip() == "true"
    if not is_worktree:
        blockers.append("not_git_worktree")

    branch = current_branch(repo) if is_worktree else None
    status = git_output(repo, ["status", "--porcelain=v1"]) if is_worktree else {"stdout": ""}
    dirty_summary = compact_lines(status.get("stdout", ""))
    head = git_output(repo, ["rev-parse", "HEAD"]) if is_worktree else {"stdout": ""}

    origin_main = (
        git_output(
            repo,
            ["ls-remote", "--exit-code", "origin", "refs/heads/main"],
            timeout_ms=timeout_ms,
        )
        if is_worktree
        else {"ok": False}
    )
    if is_worktree and not origin_main["ok"]:
        blockers.append("origin_main_unreachable")

    gh_path = shutil.which("gh")
    gh_auth = run_cmd(["gh", "auth", "status"], timeout_ms=timeout_ms) if gh_path else {"ok": False}
    if gh_path is None:
        blockers.append("gh_missing")
    elif not gh_auth["ok"]:
        blockers.append("gh_auth_unavailable")

    critic = runner_available(critic_runner) if critic_runner else None
    review_runner_results = [runner_available(runner) for runner in split_csv(review_runners)]
    headless = runner_available(headless_runner) if headless_runner else None
    if critic and not critic["available"]:
        blockers.append(f"critic_runner_missing:{critic['runner']}")
    for runner in review_runner_results:
        if not runner["available"]:
            blockers.append(f"review_runner_missing:{runner['runner']}")
    if headless and not headless["available"]:
        blockers.append(f"headless_runner_missing:{headless['runner']}")

    ticket_key = detect_ticket_key(input_value)
    branch_ticket_key = detect_ticket_key(branch)

    context_key = ticket_key or branch
    context_path = (
        str(repo / "_scratch" / "_context" / f"{context_key}.md") if context_key else None
    )
    suggested_branch = f"aryan-binazir/{ticket_key}" if ticket_key else None
    branch_setup_command = None
    if ticket_key:
        branch_setup_command = command_string(
            [
                "uv",
                "run",
                "--script",
                Path(__file__).resolve().parent / "ensure_branch.py",
                "--repo",
                repo,
                "--ticket-key",
                ticket_key,
                "--base-branch",
                "main",
            ]
        )

    tools: dict[str, Any] = {
        "gh": {
            "available": gh_path is not None,
            "path": gh_path,
            "authenticated": bool(gh_auth["ok"]),
        },
        "origin_reachable": bool(origin_main.get("ok")),
        "origin_main": {"available": bool(origin_main.get("ok"))},
        "critic_runner": critic,
        "review_runners": review_runner_results,
    }
    if headless is not None:
        tools["headless_runner"] = headless

    return {
        "ok": not blockers,
        "blockers": blockers,
        "repo": {
            "path": str(repo),
            "is_worktree": is_worktree,
            "branch": branch,
            "head": head.get("stdout", "").strip() or None,
            "branch_ticket_key": branch_ticket_key,
            "dirty": bool(dirty_summary),
            "dirty_summary": dirty_summary,
            "rule_files": rule_files(repo),
        },
        "tools": tools,
        "context": {
            "ticket_key": ticket_key,
            "suggested_key": context_key,
            "suggested_path": context_path,
            "suggested_branch": suggested_branch,
            "branch_setup_command": branch_setup_command,
            "branch_setup_creates_worktree": bool(ticket_key),
            "read_main_for_ticket_work": False,
            "base_branch": "main",
        },
        "source": {
            # A ticket-shaped key does not identify the tracker. Callers must
            # resolve Linear vs Jira with available tooling, not key format.
            "type_hint": "ticket" if ticket_key else "raw",
            "ticket_key": ticket_key,
        },
        "judgment_needed": ["dirty_worktree"] if dirty_summary and not ticket_key else [],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Collect deterministic rocket repo facts.")
    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output.")
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    parser.add_argument("--input", required=True)
    parser.add_argument("--critic-runner")
    parser.add_argument("--review-runners")
    parser.add_argument("--headless-runner")
    parser.add_argument("--timeout-ms", type=int, default=10_000)
    args = parser.parse_args()
    try:
        emit(
            repo_facts(
                args.repo,
                args.input,
                args.critic_runner,
                args.review_runners,
                args.headless_runner,
                args.timeout_ms,
            ),
            pretty=args.pretty,
        )
        return 0
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        emit({"ok": False, "failure_mode": "script_error", "error": str(exc)}, pretty=args.pretty)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
