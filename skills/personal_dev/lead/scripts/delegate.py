#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["PyYAML>=6.0.2"]
# ///
"""Run a task on the configured worker model. One command, zero agent tokens.

Resolves the worker from lead.example.yaml + lead.local.yaml (local wins),
builds the runner command with the local flag conventions, executes it, and
writes the worker's output to a report file under _scratch/implementer/.
The worker is told to keep a summary file there too, so a summary survives even
when the run is killed mid-turn. A worker's `kind` (implement | explore |
review) picks the shape of that file: implementers report what changed,
explorers and reviewers write a full findings file. Workers may fan out to
codex sub-agents (`subagent:` model/effort in config, `--subagents N` for the
count); the parent then only decomposes and synthesizes. Prints a JSON result to stdout containing that
summary and a git diff --stat, so the caller usually never needs to open the
report file. Runner-agnostic: nothing here depends on how a runner formats its
final message. Emits a heartbeat line to
stderr every 60s so long foreground runs never look dead.
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from resolve_config import deep_merge, load_yaml_file  # sibling module

DEFAULT_TIMEOUT_MS = 1_500_000  # 25 minutes
HEARTBEAT_S = 60
SUMMARY_CAP_CHARS = 1500
DIFF_STAT_CAP_LINES = 30
SCRATCH_DIR = Path("_scratch") / "implementer"
# Directories (relative to --cwd) swept of *.md older than FILE_TTL_DAYS on every run.
SWEEP_DIRS = [SCRATCH_DIR]
FILE_TTL_DAYS = 7

SUMMARY_INSTRUCTIONS = {
    "implement": """
---
Progress file (required): {path}
Create it as soon as you have a plan and overwrite it whenever your status
changes; the caller reads this file, not your chat output, and it is what
survives if you are stopped early. Final version, max 15 lines, under a
`## SUMMARY` heading: what changed, what was not done, verification results
(commands and outcomes), open questions.
""",
    "explore": """
---
Findings file (required): {path}
Read-only task: the findings file is your only output. Create it as soon as
you have a plan and overwrite it as you learn; the caller reads this file, not
your chat output, and it is what survives if you are stopped early.
No length cap: the answer, evidence as file:line references, relevant call
paths and data flow, confidence, and gaps or open questions. Finish the file
with a `## SUMMARY` section (max 15 lines): the answer, key file pointers,
confidence, gaps.
""",
    "review": """
---
Findings file (required): {path}
Read-only task: the findings file is your only output; the caller applies
fixes. Create it as soon as you have a plan and overwrite it as you learn; the
caller reads this file, not your chat output, and it is what survives if you
are stopped early. Findings only, most severe first, each with: severity, file:line,
concrete failure mode, suggested fix direction. If there are no substantive
findings, say so and name what you inspected and any residual test gaps.
Finish the file with a `## SUMMARY` section (max 15 lines): finding count by
severity, the top findings, what was inspected.
""",
}
DEFAULT_KIND = "implement"
SANDBOXES = ("workspace-write", "read-only")
WORKER_ENV = "DELEGATE_WORKER"  # set in the worker's environment; a nested delegate.py call exits with guidance

ROLE_INSTRUCTION = """
---
You are the delegated worker: this run is the delegation. Do the work directly
with your own tools, and fan out through your runtime's native sub-agents when
asked to; the delegate script and the implementer/explorer skills are the
caller's interface to you. If your runtime keeps you from writing files, put
the complete findings or progress report in your final message instead — the
caller persists it.
"""

FANOUT_INSTRUCTION = """
---
Sub-agents (required): spawn {n} sub-agent{plural}{model_note}. Split the
work into disjoint slices, one per sub-agent; wait for all, then synthesize.
The sub-agents do the heavy reading and execution — with a single sub-agent,
it still takes the bulk of the work. Your own thread does decomposition,
judgment, and the final write-up. When sub-agents edit files, give each its
own disjoint set of files.
"""


@dataclass(frozen=True)
class GitSnapshot:
    commit: str | None
    preexisting_changes: tuple[str, ...]


def sweep_stale_files(cwd: Path) -> None:
    cutoff = time.time() - FILE_TTL_DAYS * 24 * 60 * 60
    for rel in SWEEP_DIRS:
        try:
            files = list((cwd / rel).glob("*.md"))
        except OSError:
            continue
        for file in files:
            try:
                if file.stat().st_mtime < cutoff:
                    file.unlink()
            except OSError:
                pass


def git_output(cwd: Path, *argv: str, input_text: str | None = None) -> str | None:
    try:
        proc = subprocess.run(
            ["git", "-C", str(cwd), *argv],
            input=input_text, capture_output=True, text=True, timeout=10,
        )
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return None
    return proc.stdout if proc.returncode == 0 else None


def git_succeeded(cwd: Path, *argv: str) -> bool:
    return git_output(cwd, *argv) is not None


def git_changed_files(cwd: Path) -> list[str] | None:
    outputs = [
        git_output(cwd, "diff", "--name-only", "-z"),
        git_output(cwd, "diff", "--cached", "--name-only", "-z"),
        git_output(cwd, "ls-files", "--others", "--exclude-standard", "-z"),
    ]
    if any(output is None for output in outputs):
        return None
    paths = {path for output in outputs for path in output.split("\0") if path}
    return sorted(path for path in paths if not path.startswith("_scratch/"))


def git_snapshot(cwd: Path) -> GitSnapshot:
    commit = git_output(cwd, "rev-parse", "--verify", "HEAD")
    changes = git_changed_files(cwd)
    return GitSnapshot(
        commit=commit.strip() if commit else None,
        preexisting_changes=tuple(changes or ()),
    )


SUBAGENT_NAME = "delegate-sub"  # claude: name of the custom sub-agent carrying the configured model


def build_command(worker: dict[str, Any], prompt: str, subagents: int) -> list[str]:
    runner = worker.get("runner")
    model = worker.get("model")
    sandbox = worker.get("sandbox") or "workspace-write"
    if sandbox not in SANDBOXES:
        raise ValueError(f"unknown sandbox: {sandbox!r} (known: {', '.join(SANDBOXES)})")
    read_only = sandbox == "read-only"
    sub = (worker.get("subagent") or {}) if subagents else {}
    if runner == "codex":
        cmd = [
            "codex",
            "--sandbox", "read-only" if read_only else "workspace-write",
            "--ask-for-approval", "on-request",
            "-c", "approvals_reviewer=auto_review",
        ]
        if model:
            cmd += ["-m", str(model)]
        effort = worker.get("reasoning_effort")
        if effort:
            cmd += ["-c", f'model_reasoning_effort="{effort}"']
        if subagents:
            cmd += ["-c", f"agents.max_concurrent_threads_per_session={subagents}"]
            if sub.get("model"):
                cmd += ["-c", f'agents.default_subagent_model="{sub["model"]}"']
            if sub.get("reasoning_effort"):
                cmd += ["-c", f'agents.default_subagent_reasoning_effort="{sub["reasoning_effort"]}"']
        cmd += ["exec", prompt]
        return cmd
    if runner == "claude":
        cmd = ["claude", "--permission-mode", "plan" if read_only else "auto"]
        if model:
            cmd += ["--model", str(model)]
        effort = worker.get("reasoning_effort")
        if effort:
            cmd += ["--effort", str(effort)]
        if sub.get("model"):
            agent_def = {
                SUBAGENT_NAME: {
                    "description": "Sub-agent for delegated slices of the current task; use it for all fan-out.",
                    "prompt": "You handle one slice of a larger task for the parent agent. Do exactly the slice you are given and report back concisely with file:line evidence.",
                    "model": str(sub["model"]),
                }
            }
            cmd += ["--agents", json.dumps(agent_def)]
        cmd += ["-p", prompt]
        return cmd
    if runner == "cursor":
        if sub.get("model") or sub.get("reasoning_effort"):
            raise ValueError("cursor runner cannot pin a sub-agent model; drop the worker's `subagent` block or use --subagents 0")
        cmd = [
            "cursor-agent", "--print", "--trust", "--auto-review",
            "--sandbox", "enabled",
        ]
        if read_only:
            cmd += ["--mode", "plan"]
        if model:
            cmd += ["--model", str(model)]
        cmd.append(prompt)
        return cmd
    raise ValueError(f"unknown runner: {runner!r} (known: codex, claude, cursor)")


def fanout_instruction(worker: dict[str, Any], subagents: int) -> str:
    sub = worker.get("subagent") or {}
    model_note = ""
    if sub.get("model"):
        if worker.get("runner") == "claude":  # claude pins the model via --agents; effort is the runtime's choice
            model_note = f" using the `{SUBAGENT_NAME}` subagent ({sub['model']}, preconfigured; it is the agent type for every spawn)"
        else:
            effort = f" at {sub['reasoning_effort']} effort" if sub.get("reasoning_effort") else ""
            model_note = f" ({sub['model']}{effort}, preconfigured; spawn with that model)"
    return FANOUT_INSTRUCTION.format(n=subagents, plural="" if subagents == 1 else "s", model_note=model_note)


def make_worktree(base_cwd: Path) -> Path | None:
    root = git_output(base_cwd, "rev-parse", "--show-toplevel")
    if not root:
        return None
    root_path = Path(root.strip())
    path = Path(tempfile.mkdtemp(prefix=f"delegate-{root_path.name}-"))
    if not git_succeeded(root_path, "worktree", "add", "--detach", str(path), "HEAD"):
        try:
            path.rmdir()
        except OSError:
            pass
        return None
    return path


def worktree_is_clean(cwd: Path, snapshot: GitSnapshot) -> bool:
    """Return whether a worktree still exactly matches its starting snapshot."""
    commit = git_output(cwd, "rev-parse", "--verify", "HEAD")
    changes = git_changed_files(cwd)
    return bool(
        snapshot.commit
        and commit
        and commit.strip() == snapshot.commit
        and changes == []
    )


def remove_clean_worktree(base_cwd: Path, cwd: Path, snapshot: GitSnapshot) -> bool:
    if not worktree_is_clean(cwd, snapshot):
        return False
    if git_succeeded(base_cwd, "worktree", "remove", str(cwd)):
        return True
    # Recheck before force: the worker or caller may have changed it meanwhile.
    if not worktree_is_clean(cwd, snapshot):
        return False
    return git_succeeded(base_cwd, "worktree", "remove", "--force", str(cwd))


def extract_summary(text: str) -> str | None:
    idx = text.rfind("## SUMMARY")
    if idx == -1:
        return None
    body = text[idx + len("## SUMMARY"):].strip()
    return body[:SUMMARY_CAP_CHARS] or None


def diff_stat(cwd: Path, snapshot: GitSnapshot) -> str | None:
    """Changes from the starting commit, including untracked files the worker added."""
    start_commit = snapshot.commit
    baseline = start_commit or git_output(
        cwd, "hash-object", "-t", "tree", "-w", "--stdin", input_text="",
    )
    diff = git_output(cwd, "diff", "--stat", baseline.strip()) if baseline else None
    if diff is None:
        diff = git_output(cwd, "diff", "--stat", "HEAD") or git_output(cwd, "diff", "--stat") or ""
    untracked = git_output(cwd, "ls-files", "--others", "--exclude-standard") or ""
    lines = diff.strip().splitlines()
    # _scratch/ holds prompt files and scratch state by local convention; files
    # untracked before the worker started are the caller's, not the worker's.
    preexisting = set(snapshot.preexisting_changes)
    lines += [
        f"untracked: {p}"
        for p in untracked.strip().splitlines()
        if not p.startswith("_scratch/") and p not in preexisting
    ]
    if not lines:
        return None
    if len(lines) > DIFF_STAT_CAP_LINES:
        lines = lines[:DIFF_STAT_CAP_LINES] + [f"... ({len(lines) - DIFF_STAT_CAP_LINES} more lines)"]
    return "\n".join(lines)


def run_with_heartbeat(cmd: list[str], cwd: Path, timeout_s: float, report: Path) -> tuple[int | None, str, str]:
    """Run cmd, streaming output to temp files, heartbeating to stderr every 60s.

    Returns (exit_code_or_None_on_timeout, stdout, stderr).
    """
    out_path = report.with_name(report.name + ".stdout.tmp")
    err_path = report.with_name(report.name + ".stderr.tmp")
    timed_out = False
    with out_path.open("w", encoding="utf-8") as out_f, err_path.open("w", encoding="utf-8") as err_f:
        proc = subprocess.Popen(
            cmd, cwd=cwd, stdin=subprocess.DEVNULL, stdout=out_f, stderr=err_f,
            env={**os.environ, WORKER_ENV: "1"},
        )
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
    parser.add_argument("--worker", help="Worker name from config (--list shows them).")
    parser.add_argument("--list", action="store_true", help="List configured workers and what each is good at.")
    parser.add_argument("--prompt", help="Inline prompt text.")
    parser.add_argument("--prompt-file", type=Path, help="File containing the self-contained prompt.")
    parser.add_argument("--cwd", type=Path, default=Path.cwd(), help="Directory to run the worker in.")
    parser.add_argument("--worktree", action="store_true", help="Run in a fresh detached git worktree of --cwd's repo.")
    parser.add_argument("--report-file", type=Path, help="Where to write worker output. Default: <cwd>/_scratch/implementer/<ts>-<worker>.md")
    parser.add_argument("--timeout-ms", type=int, help="Override worker timeout from config.")
    parser.add_argument("--subagents", type=int, help="Number of codex sub-agents the worker must fan out to (0 = none). Default: worker's `subagents` in config, else 0.")
    parser.add_argument("--dry-run", action="store_true", help="Print the resolved command without executing.")
    args = parser.parse_args()

    def fail(msg: str) -> int:
        print(json.dumps({"ok": False, "error": msg}, indent=2))
        return 1

    if os.environ.get(WORKER_ENV):
        return fail("this process is already a delegated worker: do the work directly and fan out through your runtime's native sub-agents")
    sweep_stale_files(args.cwd)

    lead_dir = Path(__file__).resolve().parents[1]
    config = deep_merge(
        load_yaml_file(lead_dir / "lead.example.yaml"),
        load_yaml_file(lead_dir / "lead.local.yaml"),
    )
    workers = config.get("workers") or {}
    if args.list:
        print(json.dumps({n: (w or {}).get("description") for n, w in workers.items()}, indent=2))
        return 0

    if bool(args.prompt) == bool(args.prompt_file) and not args.dry_run:
        return fail("provide exactly one of --prompt or --prompt-file")
    prompt = args.prompt or (args.prompt_file.read_text(encoding="utf-8") if args.prompt_file else "<dry-run>")

    name = args.worker
    if not name:
        return fail(f"provide --worker (known: {', '.join(workers) or 'none'}; --list for descriptions)")
    worker = workers.get(name)
    if worker is None:
        return fail(f"unknown worker: {name} (known: {', '.join(workers) or 'none'})")

    kind = worker.get("kind") or DEFAULT_KIND
    if kind not in SUMMARY_INSTRUCTIONS:
        return fail(f"unknown kind: {kind!r} for worker {name} (known: {', '.join(SUMMARY_INSTRUCTIONS)})")
    subagents = args.subagents if args.subagents is not None else worker.get("subagents") or 0
    if isinstance(subagents, bool) or not isinstance(subagents, int) or subagents < 0:
        return fail(f"subagents must be a non-negative integer, got {subagents!r}")
    if worker.get("subagent") is not None and not isinstance(worker.get("subagent"), dict):
        return fail(f"worker {name}: `subagent` must be a mapping with model/reasoning_effort")

    run_id = f"{int(time.time())}-{os.getpid()}-{name}"
    summary_rel = SCRATCH_DIR / f"{run_id}.summary.md"
    full_prompt = prompt + ROLE_INSTRUCTION + SUMMARY_INSTRUCTIONS[kind].format(path=summary_rel)
    if subagents:
        full_prompt += fanout_instruction(worker, subagents)
    try:
        cmd = build_command(worker, full_prompt, subagents)
    except ValueError as exc:
        return fail(str(exc))

    timeout_s = (args.timeout_ms or worker.get("timeout_ms") or DEFAULT_TIMEOUT_MS) / 1000
    result: dict[str, Any] = {
        "worker": {"name": name, **{k: v for k, v in worker.items() if k not in ("timeout_ms", "description")}},
        "command": cmd[:-1] + ["<prompt>"],  # keep stdout readable; inline prompt lands in the report
        "kind": kind,
        "subagents": subagents,
        "timeout_s": timeout_s,
    }

    if args.dry_run:
        result["ok"] = True
        result["dry_run"] = True
        print(json.dumps(result, indent=2))
        return 0
    result["worker"] = name  # config and command are for --dry-run
    result.pop("command")

    base_cwd = args.cwd.resolve()
    cwd = base_cwd
    if args.worktree:
        worktree = make_worktree(cwd)
        if worktree is None:
            return fail("worktree creation failed")
        cwd = worktree
        result["worktree"] = str(cwd)
    result["cwd"] = str(cwd)
    snapshot = git_snapshot(cwd)
    result["preexisting_changes"] = list(snapshot.preexisting_changes)

    def clean_up_worktree() -> None:
        if not args.worktree:
            return
        removed = remove_clean_worktree(base_cwd, cwd, snapshot)
        result["worktree_removed"] = removed
        if removed:
            result.pop("worktree", None)

    report = args.report_file or base_cwd / SCRATCH_DIR / f"{run_id}.md"
    report.parent.mkdir(parents=True, exist_ok=True)
    summary_file = cwd / summary_rel
    summary_file.parent.mkdir(parents=True, exist_ok=True)
    started = time.monotonic()
    try:
        exit_code, stdout, stderr = run_with_heartbeat(cmd, cwd, timeout_s, report)
    except FileNotFoundError:
        clean_up_worktree()
        result["ok"] = False
        result["error"] = f"runner CLI not found: {cmd[0]}"
        print(json.dumps(result, indent=2))
        return 1
    if exit_code is None:
        result["timed_out"] = True
    try:
        summary_text = summary_file.read_text(encoding="utf-8").strip()
    except OSError:
        summary_text = ""
    if summary_text:
        summary, summary_source = extract_summary(summary_text) or summary_text[-SUMMARY_CAP_CHARS:], "file"
    elif extract_summary(stdout):
        summary, summary_source = extract_summary(stdout), "stdout"
        summary_file.write_text(stdout.strip() + "\n", encoding="utf-8")  # persist findings a runtime kept the worker from writing
    else:
        summary, summary_source = None, "none"
    if args.worktree and summary_file.exists():
        kept = base_cwd / summary_rel  # survives worktree cleanup
        kept.parent.mkdir(parents=True, exist_ok=True)
        kept.write_text(summary_file.read_text(encoding="utf-8"), encoding="utf-8")
        summary_file = kept

    prompt_ref = str(args.prompt_file.resolve()) if args.prompt_file else "(inline; see bottom of this report)"
    parts = [
        "# delegate report",
        "",
        f"worker: {name}",
        f"runner: {worker.get('runner')}",
        f"model: {worker.get('model')}",
        f"kind: {kind}",
        f"subagents: {subagents}",
        f"cwd: {cwd}",
        f"exit_code: {exit_code}",
        f"prompt: {prompt_ref}",
        "",
        "## summary",
        "",
        summary or "(none)",
        "",
        "## output",
        "",
        stdout.strip(),
    ]
    if stderr.strip():
        parts += ["", "## stderr", "", stderr.strip()]
    if not args.prompt_file:
        parts += ["", "## prompt (inline)", "", full_prompt]
    report.write_text("\n".join(parts) + "\n", encoding="utf-8")

    result["ok"] = exit_code == 0
    result["exit_code"] = exit_code
    result["summary"] = summary
    result["summary_source"] = summary_source
    result["summary_file"] = str(summary_file)
    if kind != "implement":
        result["summary_file_note"] = "full findings; the JSON summary is only its ## SUMMARY tail"
    result["diff_stat"] = diff_stat(cwd, snapshot)
    result["report_file"] = str(report)
    result["report_note"] = "raw runner transcript, often large; search it for the specific error or tail you need"
    result["duration_s"] = round(time.monotonic() - started, 1)
    clean_up_worktree()
    print(json.dumps(result, indent=2))
    return 0 if result["ok"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
