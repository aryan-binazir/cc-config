---
name: notion-to-obsidian-sync
description: One-way sync from the user's Notion workspace into the Ar_HQ Obsidian vault, with Notion as source of truth. Use whenever the user says anything like "sync notion to obsidian", "pull notion into obsidian", "update my vault from notion", "back up notion to obsidian", "refresh my notion notes in obsidian", "mirror notion", or otherwise asks to copy/sync/back-up Notion content into their Obsidian vault. Also trigger if the user mentions a Notion sync or backup without specifying direction — this skill owns that workflow. Do NOT trigger for general Obsidian questions (use the `obsidian` skill) or for Obsidian-to-Notion sync (explicitly out of scope here). Prefer this skill over ad-hoc MCP calls — it handles conflict resolution, attachment downloads, database representation, and manifest state that you should not try to recreate from scratch.
---

# Notion → Obsidian sync

Walks the user's entire Notion workspace via the Notion MCP, and mirrors every accessible page into the Ar_HQ Obsidian vault as Obsidian-flavored markdown.

## Non-negotiables

- **Direction**: one-way, Notion → Obsidian. Never write to Notion.
- **Source of truth**: Notion wins every conflict. Local Obsidian edits are preserved in a sidecar `<basename>_conflict.md` but never propagated back.
- **Scope**: the entire accessible workspace. No filtering by page or database in v1.
- **Isolation**: all synced content lives under `Ar_HQ/Notion/`. Nothing outside that folder is ever touched.

## Vault location

`/Users/ar/Library/Mobile Documents/iCloud~md~obsidian/Documents/Ar_HQ`

This is an iCloud-synced vault. Two consequences:

1. File mtimes can change without content changing (iCloud touches them during sync). **Always detect local edits by SHA-256 of the file bytes, never by mtime.**
2. Attachment downloads and file writes should tolerate transient iCloud replication delays — if a file appears missing right after a write, wait briefly and retry once before erroring.

## Why we write to the filesystem directly (not via `obsidian` CLI)

The `obsidian` skill routes single-file operations through the `obsidian` CLI so plugins and templates stay in the loop. This skill deliberately bypasses that: bulk writes of hundreds of files through the CLI would be slow, and Obsidian picks up filesystem changes on its next scan regardless. Attachments (binary files), folder creation, and bulk frontmatter writes are also awkward through the CLI.

After sync, if the user asks to *query* or *edit* individual synced notes, hand off to the `obsidian` skill.

## When to trigger

Trigger for phrasings like:

- "sync notion to obsidian"
- "pull my notion notes"
- "update my vault from notion"
- "mirror notion into obsidian"
- "back up my notion to obsidian"
- "refresh obsidian from notion"

Do NOT trigger for:

- Generic Obsidian questions → `obsidian` skill
- Notion-only questions (no sync intent) → call MCP tools directly
- Obsidian → Notion direction → tell the user this isn't supported and ask if they meant the reverse

## Workflow

### 1. Preflight

1. Confirm the vault directory exists. If missing, stop and tell the user — don't create it blindly.
2. Check Notion MCP authentication. If the read tools aren't loaded, authenticate first:
   - Call `mcp__claude_ai_Notion__authenticate` and follow the flow to `mcp__claude_ai_Notion__complete_authentication`.
   - After auth, use `ToolSearch` with `+notion` to discover and load the Notion read tools (typical names: `list_pages`, `get_page`, `get_page_content`, `list_databases`, `query_database`, `get_block_children` — exact names depend on the MCP server version).
3. Ask the user whether to run in **dry-run** mode (default on first invocation of a session) or **apply**. Dry-run writes a plan to `~/.cache/notion-obsidian-sync/dry-run-<timestamp>/` showing every file that would be created, updated, or marked as conflict, with the first ~40 lines of each converted body. It does NOT touch the vault or download attachments.

### 2. Load manifest

Manifest path: `Ar_HQ/.notion-sync/manifest.json`. Schema and helpers: `references/manifest.md` and `scripts/manifest_util.py`.

A missing manifest means every Notion page is treated as "new" — the first run is effectively a full initial sync.

### 3. Walk the workspace

Enumerate all accessible pages and databases via the Notion MCP. For each page, capture (without fetching block content yet):

- `id` (UUID)
- `url`
- `last_edited_time`
- `parent` (for hierarchy)
- `title`
- `properties` (for database rows — become frontmatter)

Defer block fetching until step 4's diff decides we need it — this saves heavy token spend on unchanged pages.

Walking strategy: breadth-first from the workspace root. The Notion API returns children via `get_block_children` for pages and `query_database` for database rows; recurse until the tree is exhausted. Dedupe by page ID (Notion can surface the same page via multiple paths).

### 4. Diff each page against the manifest

Per-page state machine (full detail in `references/conflict-resolution.md`):

| Notion changed since last sync | Local file changed (hash ≠ manifest) | Action |
|---|---|---|
| No | No | **Skip** |
| Yes | No | **Update** (overwrite from Notion) |
| No | Yes | **Conflict** (sidecar local, overwrite from Notion — Notion is SoT, local drift should never persist silently) |
| Yes | Yes | **Conflict** (sidecar local, overwrite from Notion) |

Special cases:

- Notion page exists, no manifest entry, no Obsidian file → **Create**
- Notion page exists, no manifest entry, Obsidian file exists with matching `notion_id` in frontmatter → **Adopt** (treat as local-changed; sidecar + overwrite)
- Notion page exists, no manifest entry, Obsidian file exists at the target path WITHOUT `notion_id` → **Collision**: write new content to `<basename>_notion.md` sidecar and flag for user. Do not overwrite untracked files.
- Notion page absent, manifest entry exists → **Orphan**: log only, don't delete. v1 has no `--prune`.
- Obsidian file with a `notion_id` but no corresponding Notion page → same as orphan.

### 5. Fetch blocks and convert

For pages needing create/update/conflict, fetch the full block tree (recursively via `get_block_children`, because Notion blocks nest). Convert via `scripts/convert_page.py`:

```bash
python scripts/convert_page.py \
  --page-json <path/to/page.json> \
  --blocks-json <path/to/blocks.json> \
  --vault-root "/Users/ar/Library/Mobile Documents/iCloud~md~obsidian/Documents/Ar_HQ" \
  --target-path "Notion/foo/bar.md"
```

The script emits the final markdown to stdout (or writes directly if `--out` given) and prints any unhandled block types to stderr so they're visible in the run summary.

Mapping reference: `references/block-mapping.md` is authoritative. If the script encounters a block type not in the reference, it wraps the raw Notion JSON in a warning callout so content is never silently dropped.

### 6. Write files and update manifest

For each page requiring a write, in order:

1. Compute the Obsidian target path under `Notion/` following `references/folder-layout.md`.
2. Create parent directories if missing.
3. If conflict detected, move existing contents to sidecar BEFORE overwriting (see `references/conflict-resolution.md` for sidecar naming).
4. Write the converted markdown. Frontmatter must include:
   - `notion_id`
   - `notion_url`
   - `last_notion_edited_time`
   - `last_synced_at`
   - `notion_parent_id` (if applicable)
   - Database row properties as additional keys (see `references/databases.md`)
5. Hash the written bytes (SHA-256) and update the manifest entry.
6. Save the manifest after each page (atomic rename). This makes the run resumable — a crash mid-sync leaves a consistent manifest.

### 7. Attachments

Internal Notion-hosted files (images, videos, PDFs, generic files) are downloaded to:

```
Ar_HQ/Notion/_attachments/<source_page_id>/<sanitized_filename>
```

and referenced with Obsidian-style embeds: `![[Notion/_attachments/<page_id>/<filename>]]`.

External URLs (http[s] not on Notion's CDN) are preserved as-is — no download.

Attachment dedupe: keyed by Notion file ID if available, else by SHA-256 of the bytes. Don't re-download a file that's already present and hash-matches the manifest entry.

### 8. Summary

Print at end:

```
Notion → Obsidian sync complete.
  Created:    N
  Updated:    N
  Conflicts:  N  (sidecars: path1, path2, ...)
  Unchanged:  N
  Orphans:    N  (notion_id present in manifest or file but page gone from Notion)
  Attachments: N downloaded, N skipped (already present)
  Unhandled block types: [type_a: count, type_b: count]  (if any)
```

If any pages failed mid-convert, list their IDs and the error. A re-run will retry them.

## Dry-run output format

Dry-run writes to `~/.cache/notion-obsidian-sync/dry-run-<ISO8601>/`:

```
plan.md                 # human-readable summary of planned actions
plan.json               # machine-readable — same data as plan.md in structured form
previews/
  <page_id>.md          # first ~40 lines of what would be written to the vault
```

The user reviews `plan.md`, then re-invokes in apply mode if satisfied.

## Error handling

- **Rate limits**: exponential backoff per call. Never abort the whole sync — each page is independent. After 3 retries on a single page, log and move on.
- **Partial failure**: manifest is saved per-page, so re-running resumes cleanly.
- **Malformed blocks**: unhandled types become fenced-JSON inside a warning callout. Never drop content silently.
- **Permissions**: if a Notion page returns 403/404 mid-walk, log and skip. The user may have lost access.
- **iCloud flakiness**: on write, if the file doesn't appear on disk within 2 seconds, wait 1 second and retry the stat. Don't retry the write — it likely succeeded.

## Key design decisions (quick reference)

- Manifest at `.notion-sync/manifest.json` inside the vault so sync state travels with the vault across machines.
- `Notion/` top-level folder isolates synced content from hand-written notes.
- Conflict sidecar naming: `<basename>_conflict.md`, or `<basename>_conflict_<timestamp>.md` if the first already exists. Never overwrite a sidecar.
- Databases become folders with an `_index.md` table view and one row-file per row (`references/databases.md`).
- Pages with children become folders with `_index.md` (Hugo-style), since Obsidian can't have a file and folder with the same name.
- Content hashing over mtime, because iCloud touches mtimes.
- No deletion in v1 — orphans are logged, not removed. A future `--prune-orphans` flag can add opt-in deletion with confirmation.

## Reference files

| File | When to read |
|---|---|
| `references/block-mapping.md` | Whenever converting Notion blocks to markdown. Authoritative mapping table. |
| `references/conflict-resolution.md` | When deciding whether a page needs a sidecar, and when a Notion page is missing. |
| `references/databases.md` | When a Notion database is encountered in the walk. |
| `references/folder-layout.md` | When computing the Obsidian target path for a Notion page. |
| `references/manifest.md` | When loading, updating, or reasoning about the sync manifest. |

## Scripts

| Script | Purpose |
|---|---|
| `scripts/manifest_util.py` | Load/save/update manifest, compute file hashes, CLI wrapper for shell use. |
| `scripts/convert_page.py` | Convert a Notion page + block tree JSON to Obsidian markdown with frontmatter. |
