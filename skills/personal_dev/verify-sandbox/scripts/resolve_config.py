#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["PyYAML>=6.0.2"]
# ///

from __future__ import annotations

import argparse
import json
import os
import shlex
import sys
from pathlib import Path
from typing import Any

import yaml


ENV_KEYS = {
    "runtime": "SBX_RUNTIME",
    "postgres_image": "SBX_POSTGRES_IMAGE",
    "postgres_version": "SBX_POSTGRES_VERSION",
    "redis_image": "SBX_REDIS_IMAGE",
    "gc_ttl_hours": "SBX_GC_TTL_HOURS",
    "require_rootless": "SBX_REQUIRE_ROOTLESS",
}


def emit(payload: dict[str, Any], pretty: bool = False) -> None:
    print(json.dumps(payload, indent=2 if pretty else None, sort_keys=pretty))


def load_yaml(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    if data is None:
        return {}
    if not isinstance(data, dict):
        raise TypeError(f"{path} must contain a YAML object")
    return data


def resolve(skill_dir: Path) -> tuple[dict[str, Any], list[Path], Path]:
    example = skill_dir / "verify-sandbox.example.yaml"
    local = skill_dir / "verify-sandbox.local.yaml"
    config = {**load_yaml(example), **load_yaml(local)}
    unknown = sorted(set(config) - set(ENV_KEYS))
    if unknown:
        raise ValueError(f"unknown config keys: {', '.join(unknown)}")

    for key, env_key in ENV_KEYS.items():
        if env_key in os.environ:
            value = os.environ[env_key]
            config[key] = yaml.safe_load(value) if key in {
                "gc_ttl_hours",
                "require_rootless",
            } else value

    if isinstance(config.get("postgres_version"), int) and not isinstance(
        config["postgres_version"], bool
    ):
        config["postgres_version"] = str(config["postgres_version"])
    for key in ("runtime", "postgres_image", "postgres_version", "redis_image"):
        if not isinstance(config.get(key), str) or not config[key]:
            raise ValueError(f"{key} must be a non-empty string")
    ttl = config.get("gc_ttl_hours")
    if not isinstance(ttl, int) or isinstance(ttl, bool) or ttl < 0:
        raise ValueError("gc_ttl_hours must be a non-negative integer")
    if not isinstance(config.get("require_rootless"), bool):
        raise ValueError("require_rootless must be a boolean")

    files = [example] + ([local] if local.exists() else [])
    return config, files, local


def shell_assignments(
    config: dict[str, Any], files: list[Path], local: Path
) -> str:
    values = {
        "RUNTIME": config["runtime"],
        "POSTGRES_IMAGE": config["postgres_image"],
        "POSTGRES_VERSION": config["postgres_version"],
        "REDIS_IMAGE": config["redis_image"],
        "DEFAULT_TTL_HOURS": config["gc_ttl_hours"],
        "REQUIRE_ROOTLESS": str(config["require_rootless"]).lower(),
        "CONFIG_FILES": ",".join(str(path) for path in files),
        "LOCAL_CONFIG": str(local),
    }
    return " ".join(f"{key}={shlex.quote(str(value))}" for key, value in values.items())


def main() -> int:
    parser = argparse.ArgumentParser(description="Resolve verify-sandbox configuration.")
    output = parser.add_mutually_exclusive_group(required=True)
    output.add_argument("--resolve", action="store_true")
    output.add_argument("--shell", action="store_true")
    parser.add_argument("--pretty", action="store_true")
    args = parser.parse_args()

    try:
        skill_dir = Path(__file__).resolve().parents[1]
        config, files, local = resolve(skill_dir)
        if args.shell:
            print(shell_assignments(config, files, local))
        else:
            emit(
                {
                    "ok": True,
                    "errors": [],
                    "config": config,
                    "files": [str(path) for path in files],
                },
                pretty=args.pretty,
            )
        return 0
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        if args.shell:
            print(f"sbx: error: {exc}", file=sys.stderr)
        else:
            emit({"ok": False, "errors": [str(exc)]}, pretty=args.pretty)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
