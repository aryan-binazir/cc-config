#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["PyYAML>=6.0.2"]
# ///
"""Run a task on the configured worker model. One command, zero agent tokens.

Resolves the worker from lead.example.yaml + lead.local.yaml (local wins),
builds the runner command with the local flag conventions, executes it, and
writes the worker's output to a report file. Prints a JSON result to stdout
containing the worker's ## SUMMARY section and a git diff --stat, so the
caller usually never needs to open the report file. Emits a heartbeat line to
stderr every 60s so long foreground runs never look dead.
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from typing import Any

from resolve_config import deep_merge, load_yaml_file  # sibling module

DEFAULT_TIMEOUT_MS = 1_500_000  # 25 minutes
HEARTBEAT_S = 60
SUMMARY_CAP_CHARS = 1500
DIFF_STAT_CAP_LINES = 30


def build_command(worker: dict[str, Any], prompt: str) -> list[str]:
    runner = worker.get("runner")
    model = worker.get("model")
    if runner == "codex":
        cmd = ["codex", "exec", "--dangerously-bypass-approvals-and-sandbox"]
        if model:
            cmd += ["-m", str(model)]
        effort = worker.get("reasoning_effort")
        if effort:
            cmd += ["-c", f'model_reasoning_effort="{effort}"']
        cmd.append(prompt)
        return cmd
    if runner == "claude":
        cmd = ["claude", "--dangerously-skip-permissions"]
        if model:
            cmd += ["--model", str(model)]
        cmd += ["-p", prompt]
        return cmd
    if runner == "cursor":
        cmd = ["cursor-agent", "-p", "-f"]
        if model:
            cmd += ["--model", str(model)]
        cmd.append(prompt)
        return cmd
    raise ValueError(f"unknown runner: {runner!r} (known: codex, claude, cursor)")


def make_worktree(base_cwd: Path) -> Path:
    root = subprocess.run(
        ["git", "-C", str(base_cwd), "rev-parse", "--show-toplevel"],
        capture_output=True, text=True, check=True,
    ).stdout.strip()
    path = Path(tempfile.mkdtemp(prefix=f"delegate-{Path(root).name}-"))
    subprocess.run(
        ["git", "-C", root, "worktree", "add", "--detach", str(path), "HEAD"],
        capture_output=True, text=True, check=True,
    )
    return path


def extract_summary(text: str) -> str | None:
    idx = text.rfind("## SUMMARY")
    if idx == -1:
        return None
    body = text[idx + len("## SUMMARY"):].strip()
    return body[:SUMMARY_CAP_CHARS] or None


def diff_stat(cwd: Path) -> str | None:
    """Staged + unstaged changes to tracked files, plus untracked files."""
    def git(*argv: str) -> str | None:
        try:
            proc = subprocess.run(
                ["git", "-C", str(cwd), *argv],
                capture_output=True, text=True, timeout=10,
            )
        except (subprocess.TimeoutExpired, FileNotFoundError):
            return None
        return proc.stdout if proc.returncode == 0 else None

    diff = git("diff", "--stat", "HEAD") or git("diff", "--stat") or ""
    untracked = git("ls-files", "--others", "--exclude-standard") or ""
    lines = diff.strip().splitlines()
    # _scratch/ holds prompt files and scratch state by local convention; not signal.
    lines += [f"untracked: {p}" for p in untracked.strip().splitlines() if not p.startswith("_scratch/")]
    if not lines:
        return None
    if len(lines) > DIFF_STAT_CAP_LINES:
        lines = lines[:DIFF_STAT_CAP_LINES] + [f"... ({len(lines) - DIFF_STAT_CAP_LINES} more lines)"]
    return "\n".join(lines)


def run_with_heartbeat(cmd: list[str], cwd: Path, timeout_s: float, capture_dir: Path) -> tuple[int | None, str, str]:
    """Run cmd, streaming output to temp files, heartbeating to stderr every 60s.

    Returns (exit_code_or_None_on_timeout, stdout, stderr).
    """
    out_path = capture_dir / "delegate-stdout.tmp"
    err_path = capture_dir / "delegate-stderr.tmp"
    timed_out = False
    with out_path.open("w", encoding="utf-8") as out_f, err_path.open("w", encoding="utf-8") as err_f:
        proc = subprocess.Popen(cmd, cwd=cwd, stdin=subprocess.DEVNULL, stdout=out_f, stderr=err_f)
        started = time.monotonic()
        next_beat = HEARTBEAT_S
        last_size = 0
        last_growth = started
        while True:
            try:
                proc.wait(timeout=1)
                break
            except subprocess.TimeoutExpired:
                now = time.monotonic()
                elapsed = now - started
                size = out_path.stat().st_size + err_path.stat().st_size
                if size > last_size:
                    last_size = size
                    last_growth = now
                if elapsed >= timeout_s:
                    proc.kill()
                    proc.wait()
                    timed_out = True
                    break
                if elapsed >= next_beat:
                    quiet_s = int(now - last_growth)
                    print(
                        f"heartbeat: {int(elapsed)}s elapsed, worker output {last_size}B, "
                        f"last new output {quiet_s}s ago, timeout at {int(timeout_s)}s",
                        file=sys.stderr, flush=True,
                    )
                    next_beat += HEARTBEAT_S
    stdout = out_path.read_text(encoding="utf-8")
    stderr = err_path.read_text(encoding="utf-8")
    out_path.unlink(missing_ok=True)
    err_path.unlink(missing_ok=True)
    return (None if timed_out else proc.returncode), stdout, stderr


def main() -> int:
    parser = argparse.ArgumentParser(description="Delegate a task to the configured worker model.")
    parser.add_argument("--tier", help="Worker tier (xhigh, high, medium). Defaults to defaults.tier.")
    parser.add_argument("--prompt", help="Inline prompt text.")
    parser.add_argument("--prompt-file", type=Path, help="File containing the self-contained prompt.")
    parser.add_argument("--cwd", type=Path, default=Path.cwd(), help="Directory to run the worker in.")
    parser.add_argument("--worktree", action="store_true", help="Run in a fresh detached git worktree of --cwd's repo.")
    parser.add_argument("--report-file", type=Path, help="Where to write worker output. Default: /tmp/delegate-<ts>.md")
    parser.add_argument("--timeout-ms", type=int, help="Override worker timeout from config.")
    parser.add_argument("--dry-run", action="store_true", help="Print the resolved command without executing.")
    args = parser.parse_args()

    def fail(msg: str) -> int:
        print(json.dumps({"ok": False, "error": msg}, indent=2))
        return 1

    if bool(args.prompt) == bool(args.prompt_file) and not args.dry_run:
        return fail("provide exactly one of --prompt or --prompt-file")
    prompt = args.prompt or (args.prompt_file.read_text(encoding="utf-8") if args.prompt_file else "<dry-run>")

    lead_dir = Path(__file__).resolve().parents[1]
    config = deep_merge(
        load_yaml_file(lead_dir / "lead.example.yaml"),
        load_yaml_file(lead_dir / "lead.local.yaml"),
    )
    tier = args.tier or (config.get("defaults") or {}).get("tier")
    worker = (config.get("workers") or {}).get(tier)
    if worker is None:
        known = ", ".join(config.get("workers") or {}) or "none"
        return fail(f"missing worker tier: {tier} (known: {known})")

    try:
        cmd = build_command(worker, prompt)
    except ValueError as exc:
        return fail(str(exc))

    timeout_s = (args.timeout_ms or worker.get("timeout_ms") or DEFAULT_TIMEOUT_MS) / 1000
    result: dict[str, Any] = {
        "tier": tier,
        "worker": {k: v for k, v in worker.items() if k != "timeout_ms"},
        "command": cmd[:-1] + ["<prompt>"],  # keep stdout readable; inline prompt lands in the report
        "timeout_s": timeout_s,
    }

    if args.dry_run:
        result["ok"] = True
        result["dry_run"] = True
        print(json.dumps(result, indent=2))
        return 0

    cwd = args.cwd.resolve()
    if args.worktree:
        try:
            cwd = make_worktree(cwd)
        except subprocess.CalledProcessError as exc:
            return fail(f"worktree creation failed: {exc.stderr.strip()}")
        result["worktree"] = str(cwd)
    result["cwd"] = str(cwd)

    report = args.report_file or Path(tempfile.gettempdir()) / f"delegate-{int(time.time())}-{tier}.md"
    started = time.monotonic()
    try:
        exit_code, stdout, stderr = run_with_heartbeat(cmd, cwd, timeout_s, report.parent)
    except FileNotFoundError:
        return fail(f"runner CLI not found: {cmd[0]}")
    if exit_code is None:
        result["timed_out"] = True

    prompt_ref = str(args.prompt_file.resolve()) if args.prompt_file else "(inline; see bottom of this report)"
    parts = [
        "# delegate report",
        "",
        f"tier: {tier}",
        f"runner: {worker.get('runner')}",
        f"model: {worker.get('model')}",
        f"cwd: {cwd}",
        f"exit_code: {exit_code}",
        f"prompt: {prompt_ref}",
        "",
        "## output",
        "",
        stdout.strip(),
    ]
    if stderr.strip():
        parts += ["", "## stderr", "", stderr.strip()]
    if not args.prompt_file:
        parts += ["", "## prompt (inline)", "", prompt]
    report.write_text("\n".join(parts) + "\n", encoding="utf-8")

    result["ok"] = exit_code == 0
    result["exit_code"] = exit_code
    result["summary"] = extract_summary(stdout)
    result["diff_stat"] = diff_stat(cwd)
    result["report_file"] = str(report)
    result["duration_s"] = round(time.monotonic() - started, 1)
    print(json.dumps(result, indent=2))
    return 0 if result["ok"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
