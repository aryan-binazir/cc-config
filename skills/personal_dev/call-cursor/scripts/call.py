#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["PyYAML>=6.0.2"]
# ///

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path
from typing import Any

import yaml


def load_yaml(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    if data is None:
        return {}
    if not isinstance(data, dict):
        raise TypeError(f"{path} must contain a YAML object")
    return data


def merge(base: dict[str, Any], override: dict[str, Any]) -> dict[str, Any]:
    return {**base, **override}


def resolve(skill_dir: Path) -> tuple[dict[str, Any], Path]:
    local_path = skill_dir / "call-cursor.local.yaml"
    config = merge(
        load_yaml(skill_dir / "call-cursor.example.yaml"),
        load_yaml(local_path),
    )
    unknown = sorted(set(config) - {"model", "timeout_ms"})
    if unknown:
        raise ValueError(f"unknown config keys: {', '.join(unknown)}")
    if not isinstance(config.get("model"), str) or not config["model"]:
        raise ValueError("model must be a non-empty string")
    timeout_ms = config.get("timeout_ms")
    if not isinstance(timeout_ms, int) or isinstance(timeout_ms, bool) or timeout_ms <= 0:
        raise ValueError("timeout_ms must be a positive integer")
    return config, local_path


def command(config: dict[str, Any], prompt: str) -> list[str]:
    return [
        "cursor-agent",
        "--print",
        "--trust",
        "--sandbox",
        "enabled",
        "--model",
        config["model"],
        prompt,
    ]


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Call Cursor using merged example and local configuration."
    )
    parser.add_argument("prompt", nargs="?")
    parser.add_argument("--model")
    parser.add_argument("--timeout-ms", type=int)
    parser.add_argument("--resolve", action="store_true")
    parser.add_argument("--pretty", action="store_true")
    args = parser.parse_args()

    try:
        skill_dir = Path(__file__).resolve().parents[1]
        config, local_path = resolve(skill_dir)
        for key, value in (
            ("model", args.model),
            ("timeout_ms", args.timeout_ms),
        ):
            if value is not None:
                config[key] = value

        if args.resolve:
            print(
                json.dumps(
                    {
                        "ok": True,
                        "config": config,
                        "local_exists": local_path.exists(),
                    },
                    indent=2 if args.pretty else None,
                    sort_keys=args.pretty,
                )
            )
            return 0
        if not args.prompt:
            parser.error("prompt is required unless --resolve is used")

        return subprocess.run(
            command(config, args.prompt),
            timeout=config["timeout_ms"] / 1000,
            check=False,
        ).returncode
    except subprocess.TimeoutExpired:
        print("Cursor call timed out", file=sys.stderr)
        return 124
    except FileNotFoundError:
        print("cursor-agent is not available", file=sys.stderr)
        return 127
    except Exception as exc:  # noqa: BLE001 - CLI boundary.
        print(json.dumps({"ok": False, "error": str(exc)}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
