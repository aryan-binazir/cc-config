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


RUNNERS = {"claude", "codex", "cursor"}
REVIEW_RUNNERS = RUNNERS | {"rocket-review"}
TRACKERS = {"jira", "linear"}
CHECKOUTS = {"branch", "worktree"}
GRILL_SKILLS = {"grill-with-docs"}


def emit(payload: dict[str, Any], pretty: bool = False) -> None:
    print(json.dumps(payload, indent=2 if pretty else None, sort_keys=pretty))


def load_yaml_file(path: Path) -> dict[str, Any]:
    import yaml

    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError(f"{path} must contain a YAML object")
    return data


def validate_runner(
    config: dict[str, Any],
    key: str,
    allowed: set[str],
    errors: list[str],
    *,
    optional: bool = False,
) -> dict[str, Any] | None:
    value = config.get(key)
    if value is None and optional:
        return None
    if not isinstance(value, dict):
        errors.append(f"{key} must be a YAML object")
        return None
    runner = value.get("runner")
    if runner not in allowed:
        errors.append(f"{key}.runner must be one of: {', '.join(sorted(allowed))}")
    return value


def validate_grill(config: dict[str, Any], errors: list[str]) -> dict[str, Any] | None:
    value = config.get("grill")
    if value is None:
        return None
    if not isinstance(value, dict):
        errors.append("grill must be a YAML object")
        return None
    if value.get("skill") not in GRILL_SKILLS:
        errors.append(f"grill.skill must be one of: {', '.join(sorted(GRILL_SKILLS))}")
    return value


def validate_config(config: dict[str, Any]) -> dict[str, Any]:
    errors: list[str] = []
    checkout = config.get("checkout")
    if checkout not in CHECKOUTS:
        errors.append(f"checkout must be one of: {', '.join(sorted(CHECKOUTS))}")

    tracker = config.get("tracker")
    if tracker not in TRACKERS:
        errors.append(f"tracker must be one of: {', '.join(sorted(TRACKERS))}")

    critic = validate_runner(config, "critic", RUNNERS, errors)
    implementer = validate_runner(config, "implementer", RUNNERS, errors, optional=True)
    grill = validate_grill(config, errors)
    review = validate_runner(config, "review", REVIEW_RUNNERS, errors)

    return {
        "ok": not errors,
        "errors": errors,
        "checkout": checkout,
        "tracker": tracker,
        "critic": critic,
        "implementer": implementer,
        "grill": grill,
        "review": review,
    }


def resolve(fast_rocket_dir: Path) -> dict[str, Any]:
    local = fast_rocket_dir / "fast-rocket.local.yaml"
    if not local.is_file():
        return {
            "ok": False,
            "failure_mode": "missing_local_config",
            "errors": [
                "fast-rocket.local.yaml is required; copy one flat configuration "
                "from fast-rocket.example.yaml"
            ],
            "fast_rocket_dir": str(fast_rocket_dir),
            "config_path": str(local),
        }

    payload = validate_config(load_yaml_file(local))
    return {
        **payload,
        "fast_rocket_dir": str(fast_rocket_dir),
        "config_path": str(local),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Resolve the required flat Fast Rocket config.")
    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output.")
    parser.add_argument(
        "--fast-rocket-dir",
        type=Path,
        default=Path(__file__).resolve().parents[1],
    )
    args = parser.parse_args()
    try:
        payload = resolve(args.fast_rocket_dir.resolve())
        emit(payload, pretty=args.pretty)
        return 0 if payload["ok"] else 1
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        emit({"ok": False, "failure_mode": "script_error", "error": str(exc)}, pretty=args.pretty)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
