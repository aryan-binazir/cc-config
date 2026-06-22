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


def emit(payload: dict[str, Any], pretty: bool = False) -> None:
    print(json.dumps(payload, indent=2 if pretty else None, sort_keys=pretty))


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def load_yaml_file(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    import yaml

    data = yaml.safe_load(read_text(path))
    if data is None:
        return {}
    if not isinstance(data, dict):
        raise ValueError(f"{path} must contain a YAML object")
    return data


def deep_merge(base: dict[str, Any], override: dict[str, Any]) -> dict[str, Any]:
    merged = dict(base)
    for key, value in override.items():
        if isinstance(value, dict) and isinstance(merged.get(key), dict):
            merged[key] = deep_merge(merged[key], value)
        else:
            merged[key] = value
    return merged


def resolve_profiles(rocket_dir: Path, plan_profile: str | None, review_profile: str | None) -> dict[str, Any]:
    example = rocket_dir / "rocket.example.yaml"
    local = rocket_dir / "rocket.local.yaml"
    config = deep_merge(load_yaml_file(example), load_yaml_file(local))
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
            review_name = review_profile or plan.get("review_profile") or review_name

    review = None
    if review_name:
        review = review_profiles.get(review_name)
        if review is None:
            errors.append(f"missing review profile: {review_name}")

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
    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output.")
    parser.add_argument("--rocket-dir", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--plan-profile")
    parser.add_argument("--review-profile")
    args = parser.parse_args()
    try:
        emit(resolve_profiles(args.rocket_dir, args.plan_profile, args.review_profile), pretty=args.pretty)
        return 0
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        emit({"ok": False, "failure_mode": "script_error", "error": str(exc)}, pretty=args.pretty)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
