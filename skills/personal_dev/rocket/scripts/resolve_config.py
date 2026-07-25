#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["PyYAML>=6.0.2"]
# ///

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

import yaml

RUNNERS = {"claude", "codex", "cursor"}
REVIEW_RUNNERS = RUNNERS | {"rocket-review"}
TRACKERS = {"jira", "linear"}
CHECKOUTS = {"branch", "worktree"}
GRILL_SKILLS = {"grill-with-docs"}
DEFAULT_TIMEOUT_MS = 1_500_000


def emit(payload: dict[str, Any], pretty: bool = False) -> None:
    print(json.dumps(payload, indent=2 if pretty else None, sort_keys=pretty))


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def load_yaml_file(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}

    data = yaml.safe_load(read_text(path))
    if data is None:
        return {}
    if not isinstance(data, dict):
        raise TypeError(f"{path} must contain a YAML object")
    return data


def deep_merge(base: dict[str, Any], override: dict[str, Any]) -> dict[str, Any]:
    merged = dict(base)
    for key, value in override.items():
        if isinstance(value, dict) and isinstance(merged.get(key), dict):
            merged[key] = deep_merge(merged[key], value)
        else:
            merged[key] = value
    return merged


def apply_timeout_defaults(config: dict[str, Any]) -> None:
    for profile in (config.get("plan_profiles") or {}).values():
        if not isinstance(profile, dict):
            continue
        for key in ("critic", "implementer", "review"):
            runner = profile.get(key)
            if isinstance(runner, dict) and runner.get("runner") in RUNNERS:
                runner.setdefault("timeout_ms", DEFAULT_TIMEOUT_MS)

    for profile in (config.get("review_profiles") or {}).values():
        if not isinstance(profile, dict):
            continue
        for reviewer in profile.get("reviewers") or []:
            if isinstance(reviewer, dict) and reviewer.get("runner") in RUNNERS:
                reviewer.setdefault("timeout_ms", DEFAULT_TIMEOUT_MS)


def validate_timeout(config: dict[str, Any], key: str, errors: list[str]) -> None:
    timeout_ms = config.get("timeout_ms")
    if (
        not isinstance(timeout_ms, int)
        or isinstance(timeout_ms, bool)
        or timeout_ms <= 0
    ):
        errors.append(f"{key}.timeout_ms must be a positive integer")


def validate_runner(
    config: dict[str, Any],
    key: str,
    allowed: set[str],
    errors: list[str],
    *,
    optional: bool = False,
) -> None:
    value = config.get(key)
    if value is None and optional:
        return
    if not isinstance(value, dict):
        errors.append(f"{key} must be a YAML object")
        return
    if value.get("runner") not in allowed:
        errors.append(f"{key}.runner must be one of: {', '.join(sorted(allowed))}")
    elif value["runner"] in RUNNERS:
        validate_timeout(value, key, errors)


def validate_plan_profile(config: dict[str, Any], errors: list[str]) -> None:
    if config.get("checkout") not in CHECKOUTS:
        errors.append(f"checkout must be one of: {', '.join(sorted(CHECKOUTS))}")
    if config.get("tracker") not in TRACKERS:
        errors.append(f"tracker must be one of: {', '.join(sorted(TRACKERS))}")

    validate_runner(config, "critic", RUNNERS, errors)
    validate_runner(config, "implementer", RUNNERS, errors, optional=True)
    validate_runner(config, "review", REVIEW_RUNNERS, errors)

    grill = config.get("grill")
    if grill is not None:
        if not isinstance(grill, dict):
            errors.append("grill must be a YAML object")
        elif grill.get("skill") not in GRILL_SKILLS:
            errors.append(
                f"grill.skill must be one of: {', '.join(sorted(GRILL_SKILLS))}"
            )


def validate_review_profile(config: dict[str, Any], errors: list[str]) -> None:
    reviewers = config.get("reviewers") or []
    for index, reviewer in enumerate(reviewers):
        if isinstance(reviewer, dict) and reviewer.get("runner") in RUNNERS:
            validate_timeout(reviewer, f"reviewers[{index}]", errors)


def resolve_profiles(
    rocket_dir: Path,
    plan_profile: str | None,
    review_profile: str | None,
) -> dict[str, Any]:
    example = rocket_dir / "rocket.example.yaml"
    local = rocket_dir / "rocket.local.yaml"
    config = deep_merge(load_yaml_file(example), load_yaml_file(local))
    apply_timeout_defaults(config)
    defaults = config.get("defaults") or {}
    plan_name = plan_profile or defaults.get("plan_profile")
    review_name = review_profile or defaults.get("review_profile")
    plan_profiles = config.get("plan_profiles") or {}
    review_profiles = config.get("review_profiles") or {}
    errors: list[str] = []

    plan = None
    if plan_name:
        plan = plan_profiles.get(plan_name)
        if plan is None:
            errors.append(f"missing plan profile: {plan_name}")
        else:
            validate_plan_profile(plan, errors)
            review_name = review_profile or plan.get("review_profile") or review_name

    review = None
    if review_name:
        review = review_profiles.get(review_name)
        if review is None:
            errors.append(f"missing review profile: {review_name}")
        else:
            validate_review_profile(review, errors)

    return {
        "ok": not errors,
        "errors": errors,
        "rocket_dir": str(rocket_dir),
        "local_exists": local.exists(),
        "defaults": defaults,
        "plan_profile": {"name": plan_name, "config": plan},
        "review_profile": {"name": review_name, "config": review},
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Resolve rocket plan/review profiles.")
    parser.add_argument(
        "profile",
        nargs="?",
        help="Optional plan profile selected by `$rocket <profile>`.",
    )
    parser.add_argument(
        "--pretty", action="store_true", help="Pretty-print JSON output."
    )
    parser.add_argument(
        "--rocket-dir", type=Path, default=Path(__file__).resolve().parents[1]
    )
    parser.add_argument(
        "--plan-profile",
        help="Legacy named form of the optional plan profile.",
    )
    parser.add_argument("--review-profile")
    args = parser.parse_args()
    try:
        if args.profile and args.plan_profile:
            parser.error("profile and --plan-profile are mutually exclusive")
        payload = resolve_profiles(
            args.rocket_dir,
            args.profile or args.plan_profile,
            args.review_profile,
        )
        emit(payload, pretty=args.pretty)
        return 0 if payload["ok"] else 1
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        emit(
            {"ok": False, "failure_mode": "script_error", "error": str(exc)},
            pretty=args.pretty,
        )
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
