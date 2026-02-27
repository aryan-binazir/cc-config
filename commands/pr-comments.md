---
description: Pull PR comments into a stable rolling checklist for discussion-first review
version: "1.1"
---

# PR Comments

Pull the currently checked-out PR's comments (all types), assign stable numbers, and maintain a rolling checklist for discussion-first review.

## Behavior

1. Identify the PR tied to the current branch.
2. Fetch all PR comment types:
   - review comments (line-level),
   - top-level issue comments,
   - review summaries.
3. Keep only currently active comments when activity metadata is available.
4. Build one combined list ordered by `createdAt` ascending.
5. Assign stable numbers and persist state in `_context/pr_reviews/pr-<number>.json`.
6. Render the full list every run with status markers:
   - open items: normal text
   - handled items: markdown strikethrough (`~~item~~`)

## Data Collection

- Use repository tooling to resolve the PR for the current branch and fetch all three comment types.
- If one endpoint does not include all comment types, combine data from multiple calls.
- Normalize each item to:
  - `id` (stable source id),
  - `type` (`review_comment`, `issue_comment`, `review_summary`),
  - `createdAt`,
  - `updatedAt` (when available),
  - `author`,
  - `body`,
  - `isActive`,
  - optional context (`path`, `line`, `commit`, `url`).

## Active-Only Filter

- Include only active comments when the source provides active/resolved/minimized/outdated metadata.
- Exclude comments marked resolved, outdated, minimized, or deleted.
- If a comment type does not expose activity state, treat returned items as active.

## State File

- Path: `_context/pr_reviews/pr-<number>.json` where `<number>` is the PR number.
- Create `_context/pr_reviews/` if missing.
- Persist structured state as JSON (not markdown).

State payload:

- `pr`: `{ number, title, url, branch }`
- `nextNumber`: integer
- `itemsById`: map keyed by normalized `id` with:
  - `number`
  - `status` (`open` or `handled`)
  - `type`
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

- Keep existing numbers from `_context/pr_reviews/pr-<number>.json` when ids match.
- Assign new numbers only to newly discovered ids, using `nextNumber`.
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
2. Numbered list in stable numeric order:
   - `1. [open] <item summary>`
   - `2. [handled] ~~<item summary>~~`
3. Final line: `Pick a number to discuss.`

For each item summary, include:

- comment type,
- author,
- short body excerpt,
- file/line when available.

## Command Intent

This command is for rolling PR-comment triage and discussion tracking. Re-running it should return the current checklist state, including previously handled items.
