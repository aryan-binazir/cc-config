#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# ///

from __future__ import annotations

import argparse
import json
import re
import subprocess
import time
from pathlib import Path
from typing import Any


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


def sanitize_path_segment(value: str) -> str:
    sanitized = re.sub(r"[^A-Za-z0-9._-]+", "-", value).strip("-")
    return sanitized or "rocket-worktree"


def default_worktree_path(repo: Path, ticket_key: str) -> Path:
    return (
        Path.home()
        / "repos"
        / ".worktrees"
        / repo.name
        / sanitize_path_segment(ticket_key)
    )


def local_branch_exists(repo: Path, branch: str) -> bool:
    result = git_output(repo, ["show-ref", "--verify", "--quiet", f"refs/heads/{branch}"])
    return bool(result["ok"])


def parse_worktrees(output: str) -> list[dict[str, str]]:
    entries: list[dict[str, str]] = []
    current: dict[str, str] = {}
    for line in output.splitlines():
        if not line.strip():
            if current:
                entries.append(current)
                current = {}
            continue
        key, _, value = line.partition(" ")
        if key == "worktree":
            current["path"] = value
        elif key == "branch":
            current["branch"] = value.removeprefix("refs/heads/")
    if current:
        entries.append(current)
    return entries


def worktree_for_branch(repo: Path, branch: str) -> Path | None:
    result = git_output(repo, ["worktree", "list", "--porcelain"])
    if not result["ok"]:
        return None
    for entry in parse_worktrees(result["stdout"]):
        if entry.get("branch") == branch and entry.get("path"):
            return Path(entry["path"])
    return None


def worktree_dirty(path: Path) -> bool:
    status = git_output(path, ["status", "--porcelain=v1"])
    return bool(status["stdout"].strip())


def fetch_latest_base(repo: Path, remote: str, base_branch: str, timeout_ms: int) -> dict[str, Any]:
    fetch = git_output(
        repo,
        [
            "fetch",
            "--prune",
            remote,
            f"refs/heads/{base_branch}:refs/remotes/{remote}/{base_branch}",
        ],
        timeout_ms=timeout_ms,
    )
    if not fetch["ok"]:
        return {
            "ok": False,
            "failure_mode": "main_unavailable",
            "remote": remote,
            "base_branch": base_branch,
            "stderr": fetch["stderr"],
        }
    ref = f"{remote}/{base_branch}"
    verify = git_output(repo, ["rev-parse", "--verify", ref])
    if not verify["ok"]:
        return {
            "ok": False,
            "failure_mode": "main_unavailable",
            "remote": remote,
            "base_branch": base_branch,
            "stderr": verify["stderr"],
        }
    return {"ok": True, "base_ref": ref, "base_head": verify["stdout"].strip()}


def detect_ticket_key(*values: str | None) -> str | None:
    for value in values:
        if not value:
            continue
        match = re.search(r"\b[A-Z][A-Z0-9]+-\d+\b", value)
        if match:
            return match.group(0)
    return None


def ensure_branch(args: argparse.Namespace) -> dict[str, Any]:
    repo = args.repo.resolve()
    worktree = git_output(repo, ["rev-parse", "--is-inside-work-tree"])
    if not worktree["ok"] or worktree["stdout"].strip() != "true":
        return {"ok": False, "failure_mode": "not_git_worktree", "repo": str(repo)}

    ticket_key = args.ticket_key or detect_ticket_key(args.input)
    if not ticket_key:
        return {
            "ok": False,
            "failure_mode": "ticket_key_required",
            "hint": "Pass --ticket-key BBA-123 or --input containing a ticket key.",
        }
    ticket_key = ticket_key.upper()
    target = args.branch_name or f"{args.prefix}/{ticket_key}"
    current = current_branch(repo)
    current_ticket = detect_ticket_key(current)

    latest_base = fetch_latest_base(repo, args.remote, args.base_branch, args.timeout_ms)
    if not latest_base["ok"]:
        return latest_base
    base_fields = {key: value for key, value in latest_base.items() if key != "ok"}

    target_worktree = worktree_for_branch(repo, target)
    if target_worktree:
        if worktree_dirty(target_worktree):
            return {
                "ok": False,
                "failure_mode": "dirty_target_worktree",
                "current_branch": current,
                "current_ticket_key": current_ticket,
                "target_branch": target,
                "ticket_key": ticket_key,
                "worktree_path": str(target_worktree),
            }
        return {
            "ok": True,
            "action": "existing_worktree",
            "branch": target,
            "worktree_path": str(target_worktree),
            **base_fields,
        }

    worktree_path = (args.worktree_path or default_worktree_path(repo, ticket_key)).resolve()
    if worktree_path.exists():
        return {
            "ok": False,
            "failure_mode": "worktree_path_exists",
            "target_branch": target,
            "ticket_key": ticket_key,
            "worktree_path": str(worktree_path),
        }
    worktree_path.parent.mkdir(parents=True, exist_ok=True)

    if current == target:
        return {
            "ok": True,
            "action": "already_on_branch",
            "branch": current,
            "worktree_path": str(repo),
            **base_fields,
        }

    if local_branch_exists(repo, target):
        added = git_output(repo, ["worktree", "add", str(worktree_path), target])
        return {
            "ok": added["ok"],
            "action": "attached_existing_branch",
            "branch": target,
            "worktree_path": str(worktree_path),
            "stderr": added["stderr"],
            **base_fields,
        }

    remote = git_output(repo, ["ls-remote", "--exit-code", args.remote, f"refs/heads/{target}"])
    if remote["ok"]:
        fetch_target = git_output(
            repo,
            [
                "fetch",
                "--prune",
                args.remote,
                f"refs/heads/{target}:refs/remotes/{args.remote}/{target}",
            ],
            timeout_ms=args.timeout_ms,
        )
        if not fetch_target["ok"]:
            return {
                "ok": False,
                "failure_mode": "target_branch_fetch_failed",
                "target_branch": target,
                "stderr": fetch_target["stderr"],
            }
        added = git_output(
            repo,
            [
                "worktree",
                "add",
                "--track",
                "-b",
                target,
                str(worktree_path),
                f"{args.remote}/{target}",
            ],
        )
        return {
            "ok": added["ok"],
            "action": "tracked_remote_worktree",
            "branch": target,
            "worktree_path": str(worktree_path),
            "stderr": added["stderr"],
            **base_fields,
        }

    created = git_output(
        repo,
        ["worktree", "add", "-b", target, str(worktree_path), latest_base["base_ref"]],
    )
    return {
        "ok": created["ok"],
        "action": "created_worktree_from_main",
        "branch": target,
        "worktree_path": str(worktree_path),
        "stderr": created["stderr"],
        **base_fields,
    }


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Create or reuse a ticket worktree from the latest remote main."
    )
    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output.")
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    parser.add_argument("--ticket-key")
    parser.add_argument("--input", default="")
    parser.add_argument("--branch-name")
    parser.add_argument("--prefix", default="aryan-binazir")
    parser.add_argument("--remote", default="origin")
    parser.add_argument("--base-branch", default="main")
    parser.add_argument("--worktree-path", type=Path)
    parser.add_argument("--timeout-ms", type=int, default=10_000)
    args = parser.parse_args()
    try:
        emit(ensure_branch(args), pretty=args.pretty)
        return 0
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        emit({"ok": False, "failure_mode": "script_error", "error": str(exc)}, pretty=args.pretty)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
