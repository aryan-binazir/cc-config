#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# ///
"""Stable, numbered checklist of active PR comments. One command, zero agent tokens.

`fetch` (default) pulls the current branch's PR via one GraphQL call, merges into
`_scratch/_pr_reviews/pr-<n>.json` (numbering, triage, and `agent` fields
survive re-runs), and prints the checklist. `resolve` records a triage
decision. `reply` posts to the right GitHub target (thread reply or PR comment)
and records the commit hash. `--json` prints the state instead of the checklist.
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

STATE_DIR = Path("_scratch/_pr_reviews")
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


def sh(*args: str, input: str | None = None) -> str:
    return subprocess.run(args, check=True, capture_output=True, text=True, input=input).stdout


def now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def fetch(pr_number: int | None) -> tuple[dict[str, Any], list[dict[str, Any]], bool]:
    """Return (pr meta, items, truncated). Items carry `active` and thread grouping."""
    owner, name = sh("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner").strip().split("/")
    number = pr_number or int(sh("gh", "pr", "view", "--json", "number", "--jq", ".number"))
    raw = sh("gh", "api", "graphql", "-f", f"query={QUERY}", "-f", f"owner={owner}", "-f", f"name={name}", "-F", f"number={number}")
    pr = json.loads(raw)["data"]["repository"]["pullRequest"]
    truncated = any(pr[k]["pageInfo"]["hasNextPage"] for k in ("comments", "reviews", "reviewThreads"))
    meta = {"number": pr["number"], "title": pr["title"], "url": pr["url"], "branch": pr["headRefName"], "repo": f"{owner}/{name}"}
    items: list[dict[str, Any]] = []

    def base(c: dict[str, Any], type_: str, active: bool) -> dict[str, Any]:
        return {"id": c["id"], "databaseId": c["databaseId"], "type": type_, "author": (c.get("author") or {}).get("login", ""),
                "body": c["body"], "createdAt": c["createdAt"], "updatedAt": c["updatedAt"], "url": c["url"], "active": active}

    for c in pr["comments"]["nodes"]:
        items.append(base(c, "issue_comment", not c["isMinimized"]))
    for r in pr["reviews"]["nodes"]:
        items.append(base(r, "review_summary", bool(r["body"].strip()) and not r["isMinimized"]) | {"state": r["state"]})
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
    return STATE_DIR / f"pr-{number}.json"


def load_state(number: int) -> dict[str, Any]:
    p = state_path(number)
    if p.exists():
        return json.loads(p.read_text())
    return {"pr": {}, "nextNumber": 1, "itemsById": {}}


def save_state(state: dict[str, Any]) -> None:
    p = state_path(state["pr"]["number"])
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(state, indent=2) + "\n")


def merge(state: dict[str, Any], meta: dict[str, Any], items: list[dict[str, Any]]) -> dict[str, Any]:
    state["pr"] = meta
    by_id = state["itemsById"]
    seen = now()
    top = sorted((i for i in items if not i.get("parentId")), key=lambda i: i["createdAt"])
    replies = sorted((i for i in items if i.get("parentId")), key=lambda i: i["createdAt"])
    for item in top + replies:  # parents first so replies find their number
        fp = f'{item["body"]}\n{item["updatedAt"]}'
        prev = by_id.get(item["id"])
        if prev is None:
            if not item["active"] or (item.get("parentId") and item["parentId"] not in by_id):
                continue  # numbers go to items seen active at least once
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


def find(state: dict[str, Any], number: str) -> tuple[str, dict[str, Any]]:
    for id_, item in state["itemsById"].items():
        if item["number"] == number:
            return id_, item
    sys.exit(f"no item numbered {number}")


def cmd_resolve(state: dict[str, Any], args: argparse.Namespace) -> None:
    _, item = find(state, args.number)
    item.update(status="handled", resolution=args.resolution, resolutionNote=" ".join(args.note) or None)


def cmd_reply(state: dict[str, Any], args: argparse.Namespace) -> None:
    _, item = find(state, args.number)
    body = args.body or f'{args.agent}: addressed in commit `{args.commit}`.'
    if args.testing:
        body += f"\n\nTesting: `{args.testing}`"
    repo, number = state["pr"]["repo"], state["pr"]["number"]
    if item["type"] == "review_comment":
        target = state["itemsById"][item["parentId"]]["databaseId"] if item.get("parentId") else item["databaseId"]
        if item.get("parentId"):
            body += f'\n\nRe: {item["url"]}'
        out = sh("gh", "api", "-X", "POST", f"repos/{repo}/pulls/{number}/comments/{target}/replies", "-f", f"body={body}")
        reply_url = json.loads(out)["html_url"]
    else:
        body += f'\n\nRe: {item["url"]}'
        reply_url = sh("gh", "pr", "comment", str(number), "--repo", repo, "--body", body).strip()
    item.update(status="handled", resolution="accepted",
                agent={"status": "handled", "agent": args.agent, "commit": args.commit, "replyUrl": reply_url, "handledAt": now()})
    print(reply_url)


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--pr", type=int, help="PR number (default: the current branch's PR)")
    ap.add_argument("--json", action="store_true", help="print state JSON instead of the checklist")
    sub = ap.add_subparsers(dest="cmd")
    sub.add_parser("fetch", help="pull, merge, print (default)")
    r = sub.add_parser("resolve", help="record a triage decision")
    r.add_argument("number")
    r.add_argument("resolution", choices=["accepted", "rejected", "deferred"])
    r.add_argument("note", nargs="*")
    p = sub.add_parser("reply", help="post a reply on GitHub and mark handled")
    p.add_argument("number")
    p.add_argument("--commit", required=True)
    p.add_argument("--agent", default="Codex")
    p.add_argument("--body", help="full reply body (default: '<agent>: addressed in commit `<hash>`.')")
    p.add_argument("--testing", help="appended as a Testing line")
    args = ap.parse_args()

    truncated = False
    if args.cmd in (None, "fetch"):
        meta, items, truncated = fetch(args.pr)
        state = merge(load_state(meta["number"]), meta, items)
    else:
        number = args.pr or int(sh("gh", "pr", "view", "--json", "number", "--jq", ".number"))
        state = load_state(number)
        if not state["pr"]:
            sys.exit(f"{state_path(number)} is missing — run fetch first")
        {"resolve": cmd_resolve, "reply": cmd_reply}[args.cmd](state, args)
    save_state(state)
    print(json.dumps(state, indent=2) if args.json else render(state, truncated))


if __name__ == "__main__":
    main()
