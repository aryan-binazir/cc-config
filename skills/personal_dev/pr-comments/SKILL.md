---
name: pr-comments
description: Pull active PR comments for the current branch, keep stable numbering in `_scratch/_pr_reviews`, and run discussion-first triage. Use when inspecting PR comments, review threads, unresolved feedback, or a rolling PR comment checklist.
---

# PR Comments

Build a stable checklist of active PR comments for the PR attached to the current branch.

## Fast Path

1. Resolve the current branch's PR.
2. Fetch PR metadata, review threads/comments, issue comments, and review summaries with one batched query when available; otherwise use the fewest calls needed.
3. For review comments, fetch thread id plus thread-level `isResolved`; it lives on the thread, not the comment.
4. Drop resolved review threads from output. Drop minimized, deleted, outdated, or inactive comments from output. Keep prior state entries. If activity state is unavailable, treat returned items as active.
5. Sort top-level items by `createdAt`; nest review-thread replies under the first thread comment. Issue comments and review summaries are top-level.
6. Merge with `_scratch/_pr_reviews/pr-<number>.json`; never renumber existing ids.
7. Render the full active list every run.

## State

Store JSON at `_scratch/_pr_reviews/pr-<number>.json`:

- `pr`: number, title, url, branch.
- `nextNumber`.
- `itemsById`, keyed by source id, with number, status (`open`/`handled`), type (`review_comment`/`issue_comment`/`review_summary`), thread id, timestamps, author, body, active/resolved state, optional path/line/commit/url, resolution fields, `lastSeenAt`, and `lastSeenFingerprint` (body + `updatedAt`).

Create `_scratch/_pr_reviews/` if missing.

This file is the single shared PR state store: the `pr-address-comments` skill
writes its handled state into an `agent` object on the same `itemsById` entries.
Preserve any `agent` fields when updating items; never delete them.

Numbering:

- Top-level items get `1`, `2`, `3`; replies get `1.1`, `1.2`.
- New top-level numbers use `nextNumber`; new replies increment the highest existing sub-number in that thread.
- Keep handled status unless auto-reopened.

Auto-reopen and clear resolution when an item becomes active again, or its `updatedAt`, body, or fingerprint changes.

## Triage

Discussion-first is mandatory. Do not start code changes before a decision on the relevant item.

For each selected number, ask whether to accept and implement, reject with rationale, or defer. Then mark it handled and persist `resolution` (`accepted`/`rejected`/`deferred`) plus a short `resolutionNote`. Stay scoped to PR comments.

## Output

Always output:

1. PR title + link.
2. Stable numbered checklist, including open and handled items. Indent replies. Show `[open]` or `[handled]`; strikethrough handled item text.
3. Final line exactly: `Pick a number to discuss.`

Each item: type, author, short excerpt, file/line if available.

## Learning Repository

Store reusable lessons from PR comments in `_scratch/_agent_notes/<topic>.md`, updating an existing topic when appropriate.
