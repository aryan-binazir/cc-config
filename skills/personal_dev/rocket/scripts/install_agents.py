#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# ///

from __future__ import annotations

import argparse
import json
import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from pathlib import Path
from typing import Any


RUNTIMES = {
    "codex": {
        "source": Path("agents/codex"),
        "target": Path(".codex/agents"),
    },
    "claude": {
        "source": Path("agents/claude"),
        "target": Path(".claude/agents"),
    },
}


@dataclass(frozen=True)
class LinkJob:
    runtime: str
    source: Path
    target: Path
    relative_path: str


def emit(payload: dict[str, Any], pretty: bool = False) -> None:
    print(json.dumps(payload, indent=2 if pretty else None, sort_keys=pretty))


def repo_root_from_script() -> Path:
    return Path(__file__).resolve().parents[4]


def selected_runtimes(value: str) -> list[str]:
    if value == "all":
        return list(RUNTIMES)
    return [value]


def same_target(link: Path, source: Path) -> bool:
    try:
        return link.resolve(strict=True) == source.resolve(strict=True)
    except FileNotFoundError:
        return False


def relative_symlink_target(source: Path, target: Path) -> str:
    return os.path.relpath(source, start=target.parent)


def iter_jobs(
    repo_root: Path,
    runtimes: list[str],
    create_target_dirs: bool,
    dry_run: bool,
) -> tuple[list[LinkJob], list[dict[str, Any]]]:
    jobs: list[LinkJob] = []
    issues: list[dict[str, Any]] = []

    for runtime in runtimes:
        config = RUNTIMES[runtime]
        source_dir = repo_root / config["source"]
        target_dir = repo_root / config["target"]

        if not source_dir.is_dir():
            issues.append(
                {
                    "runtime": runtime,
                    "status": "missing_source_dir",
                    "source_dir": str(source_dir),
                }
            )
            continue

        if target_dir.exists() and not target_dir.is_dir():
            issues.append(
                {
                    "runtime": runtime,
                    "status": "target_not_dir",
                    "target_dir": str(target_dir),
                }
            )
            continue

        if not target_dir.is_dir() and not create_target_dirs:
            issues.append(
                {
                    "runtime": runtime,
                    "status": "missing_target_dir",
                    "target_dir": str(target_dir),
                }
            )
            continue

        if not target_dir.is_dir() and dry_run:
            issues.append(
                {
                    "runtime": runtime,
                    "status": "would_create_target_dir",
                    "target_dir": str(target_dir),
                }
            )

        for source in sorted(path for path in source_dir.rglob("*") if path.is_file()):
            rel = source.relative_to(source_dir)
            jobs.append(
                LinkJob(
                    runtime=runtime,
                    source=source,
                    target=target_dir / rel,
                    relative_path=rel.as_posix(),
                )
            )

    return jobs, issues


def link_one(job: LinkJob, dry_run: bool, force: bool) -> dict[str, Any]:
    target = job.target
    source = job.source
    payload: dict[str, Any] = {
        "runtime": job.runtime,
        "path": job.relative_path,
        "source": str(source),
        "target": str(target),
    }

    if target.is_symlink():
        if same_target(target, source):
            return {**payload, "status": "already_linked"}
        if not force:
            return {**payload, "status": "conflict_existing_symlink"}
        if dry_run:
            return {**payload, "status": "would_relink_symlink"}
        target.unlink()
        target.symlink_to(relative_symlink_target(source, target))
        return {**payload, "status": "relinked_symlink"}

    if target.exists():
        return {**payload, "status": "conflict_existing_path"}

    if dry_run:
        return {**payload, "status": "would_link"}

    target.parent.mkdir(parents=True, exist_ok=True)
    target.symlink_to(relative_symlink_target(source, target))
    return {**payload, "status": "linked"}


def run(args: argparse.Namespace) -> dict[str, Any]:
    repo_root = args.repo_root.resolve()
    runtimes = selected_runtimes(args.runtime)

    for runtime in runtimes:
        target_dir = repo_root / RUNTIMES[runtime]["target"]
        if args.create_target_dirs and not args.dry_run:
            target_dir.mkdir(parents=True, exist_ok=True)

    jobs, setup_issues = iter_jobs(repo_root, runtimes, args.create_target_dirs, args.dry_run)
    results: list[dict[str, Any]] = []

    with ThreadPoolExecutor(max_workers=args.workers) as executor:
        futures = [executor.submit(link_one, job, args.dry_run, args.force) for job in jobs]
        for future in as_completed(futures):
            results.append(future.result())

    results.sort(key=lambda item: (item["runtime"], item["path"], item["status"]))
    setup_issues.sort(key=lambda item: (item["runtime"], item["status"]))

    bad_statuses = {
        "missing_source_dir",
        "missing_target_dir",
        "target_not_dir",
        "conflict_existing_symlink",
        "conflict_existing_path",
    }
    issue_count = sum(1 for item in setup_issues + results if item["status"] in bad_statuses)
    summary: dict[str, int] = {}
    for item in setup_issues + results:
        summary[item["status"]] = summary.get(item["status"], 0) + 1

    return {
        "ok": issue_count == 0,
        "dry_run": args.dry_run,
        "repo_root": str(repo_root),
        "runtimes": runtimes,
        "workers": args.workers,
        "summary": summary,
        "issues": setup_issues,
        "results": results,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Symlink repo-owned agent templates into project runtime dirs.")
    parser.add_argument("--repo-root", type=Path, default=repo_root_from_script())
    parser.add_argument("--runtime", choices=[*RUNTIMES.keys(), "all"], default="all")
    parser.add_argument("--workers", type=int, default=min(8, (os.cpu_count() or 2) + 2))
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--force", action="store_true", help="Replace existing symlinks only; never replace real files.")
    parser.add_argument("--create-target-dirs", action="store_true")
    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output.")
    args = parser.parse_args()

    if args.workers < 1:
        emit({"ok": False, "failure_mode": "invalid_workers", "error": "--workers must be >= 1"}, pretty=args.pretty)
        return 2

    try:
        payload = run(args)
        emit(payload, pretty=args.pretty)
        return 0 if payload["ok"] else 1
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        emit({"ok": False, "failure_mode": "script_error", "error": str(exc)}, pretty=args.pretty)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
