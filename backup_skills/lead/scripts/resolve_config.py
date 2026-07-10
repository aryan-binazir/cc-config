#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["PyYAML>=6.0.2"]
# ///
"""Inspect the merged lead config (lead.example.yaml + lead.local.yaml).

Shared YAML loading/merging helpers for delegate.py live here too.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


def emit(payload: dict[str, Any], pretty: bool = False) -> None:
    print(json.dumps(payload, indent=2 if pretty else None, sort_keys=pretty))


def load_yaml_file(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    import yaml

    data = yaml.safe_load(path.read_text(encoding="utf-8"))
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


def resolve(lead_dir: Path, tier: str | None) -> dict[str, Any]:
    example = lead_dir / "lead.example.yaml"
    local = lead_dir / "lead.local.yaml"
    config = deep_merge(load_yaml_file(example), load_yaml_file(local))
    defaults = config.get("defaults") or {}
    tier_name = tier or defaults.get("tier")
    workers = config.get("workers") or {}
    errors: list[str] = []

    worker = None
    if tier_name:
        worker = workers.get(tier_name)
        if worker is None:
            errors.append(f"missing worker tier: {tier_name} (known: {', '.join(workers) or 'none'})")
    else:
        errors.append("no tier given and no default tier configured")

    return {
        "ok": not errors,
        "errors": errors,
        "lead_dir": str(lead_dir),
        "local_exists": local.exists(),
        "defaults": defaults,
        "workers": sorted(workers),
        "worker": {"tier": tier_name, "config": worker},
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Inspect the merged lead config and resolve a worker tier.")
    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output.")
    parser.add_argument("--lead-dir", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--tier", help="Worker tier, e.g. xhigh, high, medium, low. Defaults to defaults.tier.")
    args = parser.parse_args()
    try:
        emit(resolve(args.lead_dir, args.tier), pretty=args.pretty)
        return 0
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        emit({"ok": False, "failure_mode": "script_error", "error": str(exc)}, pretty=args.pretty)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
