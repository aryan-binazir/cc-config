#!/usr/bin/env python3
"""Manifest read/write helpers for the notion-to-obsidian-sync skill.

Manifest location: <vault_root>/.notion-sync/manifest.json
See references/manifest.md for the schema and rationale.

CLI usage (from the skill via Bash):

    python manifest_util.py init <vault-root>
    python manifest_util.py load <vault-root>
    python manifest_util.py get-page <vault-root> <page_id>
    python manifest_util.py upsert-page <vault-root> <page_id> --json '<entry_json>'
    python manifest_util.py remove-page <vault-root> <page_id>
    python manifest_util.py hash-file <path>
    python manifest_util.py set-run-started <vault-root>
    python manifest_util.py set-run-completed <vault-root>
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
import tempfile
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

MANIFEST_VERSION = 1
MANIFEST_DIR = ".notion-sync"
MANIFEST_FILENAME = "manifest.json"


def manifest_path(vault_root: Path) -> Path:
    return vault_root / MANIFEST_DIR / MANIFEST_FILENAME


def empty_manifest() -> dict[str, Any]:
    return {
        "version": MANIFEST_VERSION,
        "last_run_started_at": None,
        "last_run_completed_at": None,
        "pages": {},
        "attachments": {},
        "unhandled_block_types": {},
    }


def load(vault_root: Path) -> dict[str, Any]:
    path = manifest_path(vault_root)
    if not path.exists():
        return empty_manifest()
    with path.open("r", encoding="utf-8") as f:
        data = json.load(f)
    # forward-compat: fill in any missing top-level keys so callers can assume shape
    for key, default in empty_manifest().items():
        data.setdefault(key, default)
    return data


def save(vault_root: Path, manifest: dict[str, Any]) -> None:
    path = manifest_path(vault_root)
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, tmp = tempfile.mkstemp(dir=path.parent, prefix=".manifest-", suffix=".json")
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump(manifest, f, indent=2, sort_keys=True, ensure_ascii=False)
            f.write("\n")
        os.replace(tmp, path)
    except Exception:
        try:
            os.unlink(tmp)
        except OSError:
            pass
        raise


def hash_bytes(data: bytes) -> str:
    return "sha256:" + hashlib.sha256(data).hexdigest()


def hash_file(path: Path) -> str | None:
    """Hash the raw bytes of a file. Returns None if the file is missing.

    Uses content only (not mtime) because iCloud touches mtimes during sync.
    """
    if not path.exists():
        return None
    # iCloud-aware: if the path is a .icloud placeholder, force-read to materialize.
    if path.suffix == ".icloud":
        return None
    with path.open("rb") as f:
        h = hashlib.sha256()
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return "sha256:" + h.hexdigest()


def now_iso() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z")


# --- CLI -------------------------------------------------------------------


def _cmd_init(args: argparse.Namespace) -> int:
    vault = Path(args.vault_root)
    path = manifest_path(vault)
    if path.exists():
        print(f"manifest already exists: {path}", file=sys.stderr)
        return 0
    save(vault, empty_manifest())
    print(str(path))
    return 0


def _cmd_load(args: argparse.Namespace) -> int:
    m = load(Path(args.vault_root))
    json.dump(m, sys.stdout, indent=2, sort_keys=True, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0


def _cmd_get_page(args: argparse.Namespace) -> int:
    m = load(Path(args.vault_root))
    entry = m["pages"].get(args.page_id)
    json.dump(entry, sys.stdout, indent=2, sort_keys=True, ensure_ascii=False)
    sys.stdout.write("\n")
    return 0


def _cmd_upsert_page(args: argparse.Namespace) -> int:
    entry = json.loads(args.json)
    vault = Path(args.vault_root)
    m = load(vault)
    m["pages"][args.page_id] = entry
    save(vault, m)
    return 0


def _cmd_remove_page(args: argparse.Namespace) -> int:
    vault = Path(args.vault_root)
    m = load(vault)
    m["pages"].pop(args.page_id, None)
    save(vault, m)
    return 0


def _cmd_hash_file(args: argparse.Namespace) -> int:
    h = hash_file(Path(args.path))
    sys.stdout.write((h or "") + "\n")
    return 0


def _cmd_set_run_started(args: argparse.Namespace) -> int:
    vault = Path(args.vault_root)
    m = load(vault)
    m["last_run_started_at"] = now_iso()
    save(vault, m)
    return 0


def _cmd_set_run_completed(args: argparse.Namespace) -> int:
    vault = Path(args.vault_root)
    m = load(vault)
    m["last_run_completed_at"] = now_iso()
    save(vault, m)
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("init"); p.add_argument("vault_root"); p.set_defaults(func=_cmd_init)
    p = sub.add_parser("load"); p.add_argument("vault_root"); p.set_defaults(func=_cmd_load)
    p = sub.add_parser("get-page"); p.add_argument("vault_root"); p.add_argument("page_id"); p.set_defaults(func=_cmd_get_page)

    p = sub.add_parser("upsert-page")
    p.add_argument("vault_root"); p.add_argument("page_id")
    p.add_argument("--json", required=True, help="JSON object for the page entry")
    p.set_defaults(func=_cmd_upsert_page)

    p = sub.add_parser("remove-page"); p.add_argument("vault_root"); p.add_argument("page_id"); p.set_defaults(func=_cmd_remove_page)
    p = sub.add_parser("hash-file"); p.add_argument("path"); p.set_defaults(func=_cmd_hash_file)
    p = sub.add_parser("set-run-started"); p.add_argument("vault_root"); p.set_defaults(func=_cmd_set_run_started)
    p = sub.add_parser("set-run-completed"); p.add_argument("vault_root"); p.set_defaults(func=_cmd_set_run_completed)

    return parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
