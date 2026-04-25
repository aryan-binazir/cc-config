# Manifest — schema and semantics

## Location

`Ar_HQ/.notion-sync/manifest.json`

The manifest lives inside the vault, inside a dot-prefixed directory so Obsidian ignores it (Obsidian skips dotfiles by default). This placement means the sync state travels with the vault across machines via iCloud — if the user syncs the vault to a new Mac, the manifest comes along and the first sync there is incremental, not a full re-download.

A sidecar location (e.g. `~/.config/notion-obsidian-sync/`) was considered and rejected: it would cause full re-syncs whenever the user moves machines or blows away their home directory.

## Schema

```json
{
  "version": 1,
  "last_run_started_at": "2026-04-23T10:00:00Z",
  "last_run_completed_at": "2026-04-23T10:04:32Z",
  "pages": {
    "<notion_page_uuid>": {
      "notion_url": "https://www.notion.so/...",
      "obsidian_path": "Notion/Projects/Acme rebrand/_index.md",
      "last_notion_edited_time": "2026-04-22T15:00:00Z",
      "last_synced_at": "2026-04-23T10:00:12Z",
      "content_hash": "sha256:abcdef1234...",
      "parent_id": "<parent_notion_page_uuid_or_null>",
      "is_database": false,
      "is_database_row": false
    }
  },
  "attachments": {
    "<notion_file_id_or_content_hash>": {
      "source_page_id": "<notion_page_uuid>",
      "obsidian_path": "Notion/_attachments/<page_id>/diagram.png",
      "content_hash": "sha256:...",
      "downloaded_at": "2026-04-23T10:00:12Z"
    }
  },
  "unhandled_block_types": {
    "<block_type_string>": <count_observed_in_last_run>
  }
}
```

## Keying

- **Pages** are keyed by Notion page UUID. UUIDs are stable across renames, moves, property edits, and workspace reorganization. Using the path would break whenever a page is renamed.
- **Attachments** are keyed by Notion file ID when available (block contains a file block with an `id`). When the file is exposed only via a signed URL without an ID, fall back to keying on the SHA-256 of the downloaded bytes. This means the same image embedded in two pages only downloads once.

## Update semantics

- **Atomic writes**: write to `manifest.json.tmp`, then `os.replace()` to the final name. Never leave a half-written manifest on disk.
- **Per-page updates**: the manifest is updated and flushed after each successfully written page, not at the end of the run. A crash mid-sync leaves a consistent state reflecting the pages completed so far, and the next run resumes from where it left off.
- **Deleting the manifest** forces a full re-sync on the next run. This is the escape hatch — if the manifest gets corrupted or out of sync with reality, delete it and re-run.

## What goes in each page entry

| Field | Purpose |
|---|---|
| `notion_url` | Quick jump link; also serves as a visible tie-break when debugging |
| `obsidian_path` | Where this page currently lives in the vault. Updated on rename/move. |
| `last_notion_edited_time` | Compared against Notion's current `last_edited_time` to detect remote changes |
| `last_synced_at` | Wall-clock time of our last write. Not used for diffing (Notion's time is authoritative), but useful for reporting |
| `content_hash` | SHA-256 of the bytes we last wrote to the Obsidian file. Compared against the current file hash to detect local edits. **This is the critical field for the conflict state machine.** |
| `parent_id` | For reconstructing the tree when the user asks about hierarchy without re-walking Notion |
| `is_database` / `is_database_row` | For knowing whether an `_index.md` table view needs regenerating when rows change |

## Detecting local changes

To decide whether the Obsidian file has been edited locally since the last sync:

```
current_hash = sha256(file_bytes)
local_changed = (current_hash != manifest.content_hash)
```

**Do not use mtime.** iCloud will touch mtimes during sync, and VSCode's "save without modifying" can also nudge mtime. Hash is the only reliable signal.

When hashing, hash the raw bytes of the file as-written, including frontmatter. Don't strip frontmatter or normalize whitespace — the frontmatter contains `last_synced_at` which is part of our write and therefore part of what we expect to read back.

## CLI

Use `scripts/manifest_util.py` from the skill. Subcommands:

```
python manifest_util.py load <vault-root>
python manifest_util.py get-page <vault-root> <page_id>
python manifest_util.py upsert-page <vault-root> <page_id> --json '<entry_json>'
python manifest_util.py remove-page <vault-root> <page_id>
python manifest_util.py hash-file <file-path>
python manifest_util.py init <vault-root>              # create empty manifest if missing
python manifest_util.py set-run-started <vault-root>
python manifest_util.py set-run-completed <vault-root>
```
