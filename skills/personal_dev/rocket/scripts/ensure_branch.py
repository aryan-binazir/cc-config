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
    status = git_output(repo, ["status", "--porcelain=v1"])
    dirty = bool(status["stdout"].strip())

    if dirty:
        return {"ok": False, "failure_mode": "dirty_worktree", "current_branch": current, "target_branch": target}
    if current == target:
        return {"ok": True, "action": "already_on_branch", "branch": current}

    local = git_output(repo, ["show-ref", "--verify", "--quiet", f"refs/heads/{target}"])
    if local["ok"]:
        switched = git_output(repo, ["switch", target])
        return {"ok": switched["ok"], "action": "switched_existing", "branch": target, "stderr": switched["stderr"]}

    remote = git_output(repo, ["show-ref", "--verify", "--quiet", f"refs/remotes/origin/{target}"])
    if remote["ok"]:
        switched = git_output(repo, ["switch", "--track", f"origin/{target}"])
        return {"ok": switched["ok"], "action": "tracked_remote", "branch": target, "stderr": switched["stderr"]}

    if current not in {"main", "master"}:
        return {
            "ok": False,
            "failure_mode": "not_on_main_for_branch_create",
            "current_branch": current,
            "current_ticket_key": current_ticket,
            "target_branch": target,
            "ticket_key": ticket_key,
        }

    created = git_output(repo, ["switch", "-c", target])
    return {"ok": created["ok"], "action": "created", "branch": target, "stderr": created["stderr"]}


def main() -> int:
    parser = argparse.ArgumentParser(description="Create or switch to a ticket branch from a safe worktree.")
    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output.")
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    parser.add_argument("--ticket-key")
    parser.add_argument("--input", default="")
    parser.add_argument("--branch-name")
    parser.add_argument("--prefix", default="aryan-binazir")
    args = parser.parse_args()
    try:
        emit(ensure_branch(args), pretty=args.pretty)
        return 0
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        emit({"ok": False, "failure_mode": "script_error", "error": str(exc)}, pretty=args.pretty)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
