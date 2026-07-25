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
ALLOWED_KEYS = {"runner", "model", "effort", "reasoning_effort", "timeout_ms"}


def load(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    if data is None:
        return {}
    if not isinstance(data, dict):
        raise TypeError(f"{path} must contain a YAML object")
    return data


def validate(config: dict[str, Any]) -> dict[str, Any]:
    unknown = sorted(set(config) - ALLOWED_KEYS)
    if unknown:
        raise ValueError(f"unknown config keys: {', '.join(unknown)}")

    runner = config.get("runner")
    if runner not in RUNNERS:
        raise ValueError(f"runner must be one of: {', '.join(sorted(RUNNERS))}")
    if not isinstance(config.get("model"), str) or not config["model"]:
        raise ValueError("model must be a non-empty string")

    if runner != "claude":
        config.pop("effort", None)
    if runner != "codex":
        config.pop("reasoning_effort", None)
    effort_key = "effort" if runner == "claude" else "reasoning_effort"
    if runner != "cursor" and (
        not isinstance(config.get(effort_key), str) or not config[effort_key]
    ):
        raise ValueError(f"{effort_key} must be a non-empty string for {runner}")

    timeout_ms = config.get("timeout_ms")
    if (
        not isinstance(timeout_ms, int)
        or isinstance(timeout_ms, bool)
        or timeout_ms <= 0
    ):
        raise ValueError("timeout_ms must be a positive integer")

    return config


def resolve(skill_dir: Path) -> dict[str, Any]:
    local = skill_dir / "ask-adversary-in-block.local.yaml"
    config = validate(
        {
            **load(skill_dir / "ask-adversary-in-block.example.yaml"),
            **load(local),
        }
    )
    return {
        "ok": True,
        "config": config,
        "call_skill": f"call-{config['runner']}",
        "local_exists": local.exists(),
    }


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Resolve the configured adversary."
    )
    parser.add_argument("--pretty", action="store_true")
    args = parser.parse_args()
    try:
        payload = resolve(Path(__file__).resolve().parents[1])
        print(
            json.dumps(
                payload,
                indent=2 if args.pretty else None,
                sort_keys=args.pretty,
            )
        )
        return 0
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        print(json.dumps({"ok": False, "error": str(exc)}))
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
