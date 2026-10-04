#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["PyYAML>=6.0.2"]
# ///
"""Inspect lead.local.yaml, or lead.example.yaml when local is absent.

Shared YAML loading helpers for delegate.py live here too.
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


def load_config(lead_dir: Path) -> dict[str, Any]:
    local = lead_dir / "lead.local.yaml"
    return load_yaml_file(local if local.exists() else lead_dir / "lead.example.yaml")


def resolve(lead_dir: Path, name: str | None) -> dict[str, Any]:
    local = lead_dir / "lead.local.yaml"
    config = load_config(lead_dir)
    workers = config.get("workers") or {}
    errors: list[str] = []

    worker = None
    if name:
        worker = workers.get(name)
        if worker is None:
            errors.append(f"unknown worker: {name} (known: {', '.join(workers) or 'none'})")

    return {
        "ok": not errors,
        "errors": errors,
        "lead_dir": str(lead_dir),
        "local_exists": local.exists(),
        "workers": sorted(workers),
        "worker": {"name": name, "config": worker},
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Inspect the selected lead config and resolve a worker tier.")
    parser.add_argument("--pretty", action="store_true", help="Pretty-print JSON output.")
    parser.add_argument("--lead-dir", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--worker", help="Worker name from config. Defaults to defaults.worker.")
    args = parser.parse_args()
    try:
        emit(resolve(args.lead_dir, args.worker), pretty=args.pretty)
        return 0
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        emit({"ok": False, "failure_mode": "script_error", "error": str(exc)}, pretty=args.pretty)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
