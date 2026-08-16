#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["PyYAML>=6.0.2"]
# ///
"""Stable, numbered checklist of active PR comments. One command, zero agent tokens.

`fetch` (default) pulls the current branch's PR via one GraphQL call, merges into
`<state_dir>/pr-<n>.json` (numbering, triage, and `agent` fields survive
re-runs), sweeps state files older than `sweep_days`, and prints the checklist.
`show` prints one item in full. `resolve` records a triage decision. `reply`
posts to the right GitHub target (thread reply or PR comment) and records the
commit hash. `--json` prints the state instead of the checklist. Every failure
exits 1 with a one-line JSON `{"ok": false, "error": ..., "hint": ...}`.

Config: pr-comments.example.yaml + pr-comments.local.yaml (local wins).
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

SKILL_DIR = Path(__file__).resolve().parents[1]
QUERY = """
query($owner:String!,$name:String!,$number:Int!){
 repository(owner:$owner,name:$name){ pullRequest(number:$number){
  number title url headRefName
  comments(first:100){pageInfo{hasNextPage} nodes{id databaseId author{login} body createdAt updatedAt url isMinimized}}
  reviews(first:100){pageInfo{hasNextPage} nodes{id databaseId author{login} body state createdAt updatedAt url isMinimized}}
  reviewThreads(first:100){pageInfo{hasNextPage} nodes{id isResolved isOutdated
   comments(first:100){nodes{id databaseId author{login} body createdAt updatedAt url path line originalLine commit{oid} isMinimized}}}}
 }}}
"""


class Fail(Exception):
    def __init__(self, error: str, hint: str = ""):
        super().__init__(error)
        self.hint = hint


def load_config() -> dict[str, Any]:
    import yaml

    cfg: dict[str, Any] = {}
    for name in ("pr-comments.example.yaml", "pr-comments.local.yaml"):
        p = SKILL_DIR / name
        if p.exists():
            data = yaml.safe_load(p.read_text()) or {}
            if not isinstance(data, dict):
                raise Fail(f"{p} must contain a YAML object")
            cfg |= data
    if cfg.get("provider", "github") != "github":
        raise Fail(f'provider {cfg["provider"]!r} has no adapter yet', "github (github.com + GHES via host:) is the supported provider")
    cfg.setdefault("state_dir", "_scratch/pr_reviews")
    cfg.setdefault("sweep_days", 7)
    cfg.setdefault("agent", "Codex")
    return cfg


def gh(*args: str) -> str:
    env = os.environ | ({"GH_HOST": CFG["host"]} if CFG.get("host") else {})
    proc = subprocess.run(["gh", *args], capture_output=True, text=True, env=env)
    if proc.returncode:
        err = proc.stderr.strip() or proc.stdout.strip() or f"gh exited {proc.returncode}"
        low = err.lower()
        hint = ("run `gh auth login`" if "auth" in low else "pass --pr <number>" if "no pull requests found" in low
                else "run inside a git repo with a GitHub remote" if "git" in low else "")
        raise Fail(f"gh {args[0]}: {err.splitlines()[-1]}", hint)
    return proc.stdout


def now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def current_pr() -> int:
    return int(gh("pr", "view", "--json", "number", "--jq", ".number"))


def fetch(pr_number: int | None) -> tuple[dict[str, Any], list[dict[str, Any]], bool]:
    """Return (pr meta, items, truncated). Items carry `active` and thread grouping."""
    owner, name = gh("repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").strip().split("/")
    number = pr_number or current_pr()
    raw = json.loads(gh("api", "graphql", "-f", f"query={QUERY}", "-f", f"owner={owner}", "-f", f"name={name}", "-F", f"number={number}"))
    if raw.get("errors"):
        raise Fail("graphql: " + "; ".join(e.get("message", "?") for e in raw["errors"]), f"is #{number} a PR in {owner}/{name}?")
    pr = raw["data"]["repository"]["pullRequest"]
    truncated = any(pr[k]["pageInfo"]["hasNextPage"] for k in ("comments", "reviews", "reviewThreads"))
    meta = {"number": pr["number"], "title": pr["title"], "url": pr["url"], "branch": pr["headRefName"], "repo": f"{owner}/{name}"}
    items: list[dict[str, Any]] = []

    def base(c: dict[str, Any], type_: str, active: bool) -> dict[str, Any]:
        return {"id": c["id"], "databaseId": c["databaseId"], "type": type_, "author": (c.get("author") or {}).get("login") or "ghost",
                "body": c["body"] or "", "createdAt": c["createdAt"], "updatedAt": c["updatedAt"], "url": c["url"], "active": active}

    for c in pr["comments"]["nodes"]:
        items.append(base(c, "issue_comment", not c["isMinimized"]))
    for r in pr["reviews"]["nodes"]:
        items.append(base(r, "review_summary", bool((r["body"] or "").strip()) and not r["isMinimized"]) | {"state": r["state"]})
    for t in pr["reviewThreads"]["nodes"]:
        thread_active = not (t["isResolved"] or t["isOutdated"])
        for i, c in enumerate(t["comments"]["nodes"]):
            item = base(c, "review_comment", thread_active and not c["isMinimized"])
            item |= {"threadId": t["id"], "resolved": t["isResolved"], "outdated": t["isOutdated"], "path": c["path"],
                     "line": c["line"] or c["originalLine"], "commit": (c.get("commit") or {}).get("oid"),
                     "parentId": None if i == 0 else t["comments"]["nodes"][0]["id"]}
            items.append(item)
    return meta, items, truncated


def state_path(number: int) -> Path:
    return Path(CFG["state_dir"]) / f"pr-{number}.json"


def load_state(number: int) -> dict[str, Any]:
    p = state_path(number)
    if not p.exists():
        return {"pr": {}, "nextNumber": 1, "itemsById": {}}
    try:
        return json.loads(p.read_text())
    except json.JSONDecodeError as exc:
        raise Fail(f"{p} is corrupt: {exc}", "delete it and run fetch again") from exc


def save_state(state: dict[str, Any]) -> None:
    p = state_path(state["pr"]["number"])
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(state, indent=2) + "\n")


def sweep(days: int) -> None:
    cutoff = time.time() - days * 86400
    for p in Path(CFG["state_dir"]).glob("pr-*.json"):
        if p.stat().st_mtime < cutoff:
            p.unlink()


def merge(state: dict[str, Any], meta: dict[str, Any], items: list[dict[str, Any]]) -> dict[str, Any]:
    state["pr"] = meta
    by_id = state["itemsById"]
    own_replies = {(v.get("agent") or {}).get("replyUrl") for v in by_id.values()}
    seen = now()
    top = sorted((i for i in items if not i.get("parentId")), key=lambda i: i["createdAt"])
    replies = sorted((i for i in items if i.get("parentId")), key=lambda i: i["createdAt"])
    for item in top + replies:  # parents first so replies find their number
        fp = f'{item["body"]}\n{item["updatedAt"]}'
        prev = by_id.get(item["id"])
        if prev is None:
            unnumbered = not item["active"] or item["url"] in own_replies or (item.get("parentId") and item["parentId"] not in by_id)
            if unnumbered:
                continue  # numbers go to items seen active at least once; own replies stay out
            if item.get("parentId"):
                parent_num = by_id[item["parentId"]]["number"]
                subs = [int(v["number"].split(".")[1]) for v in by_id.values() if v["number"].startswith(parent_num + ".")]
                number = f"{parent_num}.{max(subs, default=0) + 1}"
            else:
                number = str(state["nextNumber"])
                state["nextNumber"] += 1
            prev = by_id[item["id"]] = {"number": number, "status": "open"}
        elif prev.get("lastSeenFingerprint") != fp or (item["active"] and not prev.get("active", True)):
            prev.update(status="open", resolution=None, resolutionNote=None)
        prev.update(item, lastSeenAt=seen, lastSeenFingerprint=fp)
    for id_ in set(by_id) - {i["id"] for i in items}:  # deleted upstream
        by_id[id_]["active"] = False
    return state


def sort_key(number: str) -> tuple[int, ...]:
    return tuple(int(x) for x in number.split("."))


def render(state: dict[str, Any], truncated: bool = False) -> str:
    pr = state["pr"]
    lines = [f'PR #{pr["number"]}: {pr["title"]} <{pr["url"]}>']
    if truncated:
        lines.append("(over 100 comments/threads — showing the first page)")
    for item in sorted(state["itemsById"].values(), key=lambda i: sort_key(i["number"])):
        if not item.get("active", True):
            continue
        indent = "  " if "." in item["number"] else ""
        loc = f' {item["path"]}:{item["line"]}' if item.get("path") else ""
        excerpt = " ".join(item["body"].split())[:120]
        text = f'{item["type"]} @{item["author"]}{loc} — {excerpt}'
        if item["status"] == "handled":
            text = f'~~{text}~~ ({item.get("resolution") or (item.get("agent") or {}).get("status", "handled")})'
        lines.append(f'{indent}{item["number"]}. [{item["status"]}] {text}')
    lines.append("Pick a number to discuss.")
    return "\n".join(lines)


def find(state: dict[str, Any], number: str) -> dict[str, Any]:
    for item in state["itemsById"].values():
        if item["number"] == number:
            return item
    raise Fail(f"no item numbered {number}", "run fetch to see current numbers")


def cmd_show(state: dict[str, Any], args: argparse.Namespace) -> str:
    item = find(state, args.number)
    loc = f' {item["path"]}:{item["line"]}' if item.get("path") else ""
    head = f'{item["number"]}. [{item["status"]}] {item["type"]} @{item["author"]}{loc}\n{item["url"]}'
    if item.get("resolution"):
        head += f'\nresolution: {item["resolution"]} — {item.get("resolutionNote") or ""}'
    return f'{head}\n\n{item["body"]}'


def cmd_resolve(state: dict[str, Any], args: argparse.Namespace) -> str:
    find(state, args.number).update(status="handled", resolution=args.resolution, resolutionNote=" ".join(args.note) or None)
    return render(state)


def cmd_reply(state: dict[str, Any], args: argparse.Namespace) -> str:
    item = find(state, args.number)
    agent = args.agent or CFG["agent"]
    body = args.body or f'{agent}: addressed in commit `{args.commit}`.'
    if args.testing:
        body += f"\n\nTesting: `{args.testing}`"
    repo, number = state["pr"]["repo"], state["pr"]["number"]
    if item["type"] == "review_comment":
        parent = state["itemsById"].get(item["parentId"]) if item.get("parentId") else None
        target = (parent or item)["databaseId"]
        if parent:
            body += f'\n\nRe: {item["url"]}'
        out = gh("api", "-X", "POST", f"repos/{repo}/pulls/{number}/comments/{target}/replies", "-f", f"body={body}")
        reply_url = json.loads(out)["html_url"]
    else:
        body += f'\n\nRe: {item["url"]}'
        reply_url = gh("pr", "comment", str(number), "--repo", repo, "--body", body).strip()
    item.update(status="handled", resolution="accepted",
                agent={"status": "handled", "agent": agent, "commit": args.commit, "replyUrl": reply_url, "handledAt": now()})
    return reply_url


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--pr", type=int, help="PR number (default: the current branch's PR)")
    ap.add_argument("--json", action="store_true", help="print state JSON instead of the checklist")
    sub = ap.add_subparsers(dest="cmd")
    sub.add_parser("fetch", help="pull, merge, sweep, print (default)")
    sub.add_parser("show", help="print one item in full").add_argument("number")
    r = sub.add_parser("resolve", help="record a triage decision")
    r.add_argument("number")
    r.add_argument("resolution", choices=["accepted", "rejected", "deferred"])
    r.add_argument("note", nargs="*")
    p = sub.add_parser("reply", help="post a reply on GitHub and mark handled")
    p.add_argument("number")
    p.add_argument("--commit", required=True)
    p.add_argument("--agent", help="reply label (default: config `agent`)")
    p.add_argument("--body", help="full reply body (default: '<agent>: addressed in commit `<hash>`.')")
    p.add_argument("--testing", help="appended as a Testing line")
    args = ap.parse_args()

    if args.cmd in (None, "fetch"):
        meta, items, truncated = fetch(args.pr)
        state = merge(load_state(meta["number"]), meta, items)
        sweep(CFG["sweep_days"])
        out = render(state, truncated)
    else:
        state = load_state(args.pr or current_pr())
        if not state["pr"]:
            raise Fail("state file is missing", "run fetch first")
        out = {"show": cmd_show, "resolve": cmd_resolve, "reply": cmd_reply}[args.cmd](state, args)
    save_state(state)
    print(json.dumps(state, indent=2) if args.json else out)


if __name__ == "__main__":
    try:
        CFG = load_config()
        main()
    except Fail as exc:
        print(json.dumps({"ok": False, "error": str(exc), "hint": exc.hint}))
        sys.exit(1)
    except Exception as exc:  # noqa: BLE001 - CLI boundary
        print(json.dumps({"ok": False, "error": f"{type(exc).__name__}: {exc}", "hint": "unexpected; report with the command you ran"}))
        sys.exit(1)
