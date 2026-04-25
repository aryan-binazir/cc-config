# Conflict resolution

Notion is the source of truth. Local Obsidian edits are never propagated to Notion, and they are never silently overwritten — they are preserved in a sidecar file.

## The conflict signal

For each page with a manifest entry, compute two booleans:

- **`notion_changed`** = `notion.last_edited_time > manifest.last_synced_at`
- **`local_changed`** = `sha256(current_file_bytes) != manifest.content_hash`

These are independent.

## Decision table

| `notion_changed` | `local_changed` | Action | Sidecar? |
|---|---|---|---|
| False | False | Skip | No |
| True  | False | Overwrite with Notion content | No |
| False | True  | Sidecar + overwrite with Notion content | **Yes** |
| True  | True  | Sidecar + overwrite with Notion content | **Yes** |

The `local_changed ∧ ¬notion_changed` case (row 3) might look surprising — the local file changed but Notion didn't, why not just preserve the local edit? Because the user declared Notion as source of truth. Any local drift is, by definition, *not* the desired state. We preserve the local content in a sidecar so nothing is lost, but the canonical file reflects Notion.

This keeps the invariant simple: **the canonical Obsidian file always reflects Notion as of the last successful sync.** The user can trust that.

## Pages without a manifest entry

These represent first-time encounters or manifest resets. The behavior depends on the Obsidian side:

| Obsidian state | Action |
|---|---|
| No file at target path | **Create** — write the new file, add manifest entry |
| File exists at target path, `notion_id` frontmatter matches | **Adopt** — treat as local-changed (sidecar + overwrite). This covers re-runs after manifest deletion, or users who previously copied content from Notion manually. |
| File exists at target path, no `notion_id` or mismatched `notion_id` | **Collision** — don't touch the file. Write the Notion content to `<basename>_notion.md` and flag in the summary for user review. |

The collision case is defensive: never overwrite a file we don't recognize as ours.

## Sidecar naming

First conflict for a given canonical file:

```
foo.md           → foo_conflict.md
foo/_index.md    → foo/_index_conflict.md
```

If a `<basename>_conflict.md` already exists (from a prior unresolved conflict), append an ISO-ish timestamp:

```
foo_conflict.md → foo_conflict_20260423T100012.md
```

Timestamp is UTC, no colons (filesystem-safe), second precision.

**Never overwrite an existing sidecar.** The whole point is to preserve every local edit across sync cycles. Stacking timestamped sidecars is acceptable — the user can clean them up after reviewing.

## Sidecar frontmatter

Each sidecar gets a minimal header so the user can tell at a glance why it exists:

```markdown
---
notion_id: <original_notion_page_uuid>
notion_url: <original_notion_url>
conflict_reason: "Notion and Obsidian both changed since last sync"
conflict_detected_at: 2026-04-23T10:00:12Z
original_obsidian_path: Notion/Projects/Acme rebrand/_index.md
---

<original obsidian file contents, verbatim below>
```

Valid `conflict_reason` values:

- `"Notion and Obsidian both changed since last sync"`
- `"Obsidian changed since last sync, Notion did not"`
- `"Pre-existing Obsidian content adopted"` (for the adopt case)

Keep the rest of the original file body verbatim — including its original frontmatter. Don't try to merge frontmatters; the user wants to see exactly what they had.

## Orphans

A manifest entry whose Notion page is no longer accessible (deleted, moved out of the authenticated user's access, or permissions revoked) is an orphan.

v1 behavior:

- Leave the Obsidian file untouched
- Keep the manifest entry so we keep detecting the orphan on subsequent runs
- Log in the summary: `Orphans: 3 (path1, path2, path3)`

A future `--prune-orphans` flag can delete orphan files after user confirmation. Not in v1 — accidental deletion is worse than stale files, and the user can clean up manually after reviewing the log.

## Why "Notion always wins" is non-negotiable

Ar's design intent: Obsidian is a mirror/backup. If we ever let Obsidian edits "win" a conflict, the next sync would silently un-do them from Notion's perspective, and the user would have to know which direction won when. The one-way rule keeps mental overhead zero — if you want something to persist, change it in Notion.

## iCloud edge case

iCloud Drive occasionally touches file mtimes without modifying content (during sync, during quota rebalance, during cross-device propagation). This is why the `local_changed` check uses content hash, not mtime.

Less common but worth knowing: iCloud can also create `.icloud` placeholder files for locally-offloaded content. If the skill encounters a `.icloud` file at a path where it expects real content, force-download by reading and retrying; don't treat the placeholder as the real file.

## Dry-run behavior

In dry-run, no files are written, but the conflict decision table is still evaluated. The plan output shows, for each page:

```
ACTION: conflict
PATH:   Notion/Projects/Acme rebrand/_index.md
SIDECAR: Notion/Projects/Acme rebrand/_index_conflict.md
REASON: Notion and Obsidian both changed since last sync
NOTION_EDITED: 2026-04-22T15:00:00Z
LAST_SYNC:     2026-04-20T09:00:00Z
LOCAL_HASH:    sha256:new...
MANIFEST_HASH: sha256:old...
```

This lets the user audit conflict decisions before they're committed.
