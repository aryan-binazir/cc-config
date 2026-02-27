---
description: Pull PR comments into a stable rolling checklist for discussion-first review
version: "1.2"
---

# PR Comments

Pull the currently checked-out PR's comments (all types), assign stable numbers, and maintain a rolling checklist for discussion-first review.

## Behavior

1. Identify the PR tied to the current branch.
2. Fetch all PR comment types:
   - review comments (line-level), including thread structure,
   - top-level issue comments,
   - review summaries.
3. Group review comments by thread (comments in the same review thread share a thread id). Issue comments and review summaries are always top-level.
4. Exclude entire threads marked as resolved at the thread level (see Active-Only Filter).
5. Keep only currently active comments from remaining threads.
6. Build one combined list ordered by `createdAt` of the top-level item ascending.
7. Assign stable hierarchical numbers and persist state in `_context/pr_reviews/pr-<number>.json`.
8. Render the full list every run with status markers:
   - open items: normal text
   - handled items: markdown strikethrough (`~~item~~`)

## Data Collection

- Use repository tooling to resolve the PR for the current branch and fetch all three comment types.
- For review comments, also fetch thread-level metadata (thread id and `isResolved` state). GitHub exposes `isResolved` on the review thread object, not on individual comments.
- If one endpoint does not include all comment types, combine data from multiple calls.
- Normalize each item to:
  - `id` (stable source id),
  - `type` (`review_comment`, `issue_comment`, `review_summary`),
  - `threadId` (for review comments; null for issue comments and review summaries),
  - `createdAt`,
  - `updatedAt` (when available),
  - `author`,
  - `body`,
  - `isActive`,
  - `threadIsResolved` (for review comments; derived from the thread's `isResolved` field),
  - optional context (`path`, `line`, `commit`, `url`).

## Active-Only Filter

- For review comments: first check the **thread-level** `isResolved` field. If a thread is resolved, exclude **all** comments in that thread. Within unresolved threads, still exclude individual comments that are minimized, deleted, or outdated.
- For other comment types: exclude items marked outdated, minimized, or deleted.
- If a comment type does not expose activity state, treat returned items as active.

## State File

- Path: `_context/pr_reviews/pr-<number>.json` where `<number>` is the PR number.
- Create `_context/pr_reviews/` if missing.
- Persist structured state as JSON (not markdown).

State payload:

- `pr`: `{ number, title, url, branch }`
- `nextNumber`: integer (next top-level number)
- `itemsById`: map keyed by normalized `id` with:
  - `number` (string: `"1"` for top-level, `"1.1"` for first reply in thread 1)
  - `status` (`open` or `handled`)
  - `type`
  - `threadId` (null for non-threaded items)
  - `createdAt`
  - `updatedAt` (optional)
  - `author`
  - `body`
  - `isActive`
  - optional context fields
  - `resolution` (optional: `accepted`, `rejected`, `deferred`)
  - `resolutionNote` (optional short rationale)
  - `lastSeenAt`
  - `lastSeenFingerprint` (hash of relevant fields such as body + updatedAt)

## Stable Numbering Rules

- Top-level items (issue comments, review summaries, first comment in a review thread) get whole numbers: `1`, `2`, `3`…
- Replies within a review thread get sub-numbers under their thread's top-level number: `1.1`, `1.2`, `1.3`…
- Keep existing numbers from `_context/pr_reviews/pr-<number>.json` when ids match.
- Assign new top-level numbers using `nextNumber`. Assign new sub-numbers by incrementing the highest existing sub-number in that thread.
- Never renumber existing items.
- Keep handled status unless explicitly reopened.

## Auto-Reopen Rules

- Auto-reopen an item (`status = open`, clear `resolution` and `resolutionNote`) when any of the following happens:
  - the comment reappears as active after being inactive,
  - `updatedAt` changes,
  - `body` changes,
  - fingerprint differs from `lastSeenFingerprint`.

## Discussion-First Rules

- Discussion-first mode is mandatory. Do not jump straight to code changes.
- For each comment, ask whether to:
  - accept and implement,
  - reject with rationale,
  - defer.
- When the user decides one of the three outcomes for a numbered item, mark it `handled` and persist:
  - `resolution` = `accepted` | `rejected` | `deferred`
  - `resolutionNote` = short user-provided rationale or plan.
- Keep focus strictly on PR comments. Avoid unrelated refactors.
- Never mention tool/vendor names in user-facing output.
- Never sign responses with any product/assistant name.

## Output Format

Always output:

1. PR title + link.
2. Numbered list in stable numeric order. Indent replies under their parent thread item.
3. Final line: `Pick a number to discuss.`

For each item summary, include: comment type, author, short body excerpt, file/line when available.

### Example

```
**Add retry logic to API client** (#142)

1. [open] review — @alice: "extractRetry should back off exponentially" · src/api/client.ts:45
   1.1. [open] review — @bob: "Agreed, also cap at 30s" · src/api/client.ts:45
   1.2. [handled] ~~review — @alice: "What about jitter?" · src/api/client.ts:45~~
2. [open] comment — @carol: "Can we add a test for the timeout path?"
3. [handled] ~~review — @dave: "Nit: unused import" · src/api/client.ts:3~~

Pick a number to discuss.
```

## Learning Repository

When a PR comment teaches something worth remembering — a pattern, a gotcha, a domain rule, a better approach — capture it in `_context/agent_notes/` as a topic file (e.g., `_context/agent_notes/error-handling.md`, `_context/agent_notes/api-pagination.md`). These files are for agent consumption, not human readability.

- Before creating a new file, search `_context/agent_notes/` for an existing file on the same topic. Update it if one exists.
- File naming: lowercase, hyphenated, descriptive of the topic (not the PR or branch).
- Content and structure are at your discretion. Optimize for usefulness as future context when working on related tickets.
- This is opt-in by judgment — only write when there's a genuine takeaway, not for every comment.

## Command Intent

This command is for rolling PR-comment triage and discussion tracking. Re-running it should return the current checklist state, including previously handled items.
