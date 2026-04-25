# Folder layout

All synced content lives under `Ar_HQ/Notion/`. Nothing outside this folder is ever read or written by the sync.

## The mapping rule

A Notion page becomes:

- `Notion/<sanitized_title>.md` if it has no child pages/databases
- `Notion/<sanitized_title>/_index.md` if it has children, and each child gets its own file/folder nested inside

Hugo-style `_index.md` is used because Obsidian cannot have a file and folder with the same base name. The `_index.md` convention is familiar to users who have worked with static site generators and doesn't clash with any Obsidian convention.

## Worked example

Notion tree:

```
Workspace root
├── Projects (page, has children)
│   ├── Acme rebrand (page, has children)
│   │   └── Meeting notes (leaf page)
│   └── Q2 planning (database)
│       ├── Ship docs (row)
│       └── Talk to legal (row)
└── Personal (page, has children)
    └── Reading list (leaf page)
```

Obsidian layout:

```
Ar_HQ/Notion/
├── Projects/
│   ├── _index.md                        ← "Projects" page body
│   ├── Acme rebrand/
│   │   ├── _index.md                    ← "Acme rebrand" page body
│   │   └── Meeting notes.md
│   └── Q2 planning/
│       ├── _index.md                    ← database table view
│       ├── Ship docs.md
│       └── Talk to legal.md
└── Personal/
    ├── _index.md
    └── Reading list.md
```

## Title sanitization

Filenames must be safe on macOS (which the vault runs on) and safe in iCloud (which has some extra restrictions). Transform titles by:

1. Replace `/`, `\`, `:`, `*`, `?`, `"`, `<`, `>`, `|` with `-` (single char).
2. Collapse runs of whitespace to a single space.
3. Strip leading/trailing whitespace and dots.
4. Truncate to 180 characters (leaving headroom under the 255-byte macOS limit, accounting for UTF-8 expansion).
5. If the result is empty (title was all punctuation), fall back to the Notion page UUID.

Do NOT lowercase. Do NOT replace spaces with underscores or hyphens. Obsidian handles spaces in filenames fine, and preserving the user's title casing makes the vault more readable.

## Handling renames and moves in Notion

When a Notion page is renamed or moved:

1. The manifest entry's `obsidian_path` no longer matches the computed target.
2. The sync detects the mismatch and **moves** the file (and recursively the folder, if it has children) to the new location.
3. The manifest is updated with the new `obsidian_path`.
4. No redirect file is left at the old location.

This simplicity matters: leaving stale redirects breeds orphans, and wikilinks in Obsidian are resolved by filename not by path — so a rename that keeps the filename won't break `[[Wikilinks]]` anyway.

When a page is moved to a new parent AND renamed simultaneously, both operations apply. The file moves to its new folder and gets renamed.

## Conflicts with existing hand-written files

The `Notion/` folder is reserved for sync. If the user happens to have a hand-written file at `Ar_HQ/Notion/something.md` that doesn't have a `notion_id` in frontmatter, the sync will:

1. Refuse to overwrite it.
2. Write the Notion content to `something_notion.md` as a sidecar.
3. Flag it in the summary.

This means: **don't hand-write files under `Notion/`.** That folder is owned by the sync. If the user wants Obsidian notes near their Notion content, put them elsewhere in the vault.

## Attachments folder

`Ar_HQ/Notion/_attachments/<source_notion_page_id>/<sanitized_filename>`

Grouped by the Notion page ID of the page that references the file, not by attachment file ID. Rationale: most users think "the image from my meeting notes page", not "attachment 4f3a-abc...". Grouping by page makes the attachments folder navigable.

If the same file is referenced from multiple pages, it's downloaded once (keyed by Notion file ID in the manifest) and embedded via absolute vault path from both places. The physical location is under whichever page's ID we saw first.

Sanitization of attachment filenames follows the same rules as page titles, but with the extension preserved:

```
"My Cool Diagram (v2).png" → "My Cool Diagram (v2).png"
"weird/name?.pdf"          → "weird-name-.pdf"
```

Parentheses and other filename-safe characters are left alone — only the forbidden ones are replaced.

## What happens to the rest of the vault

Outside `Notion/` and `.notion-sync/`, the sync never reads or writes. The user's hand-written notes in `bahai/`, `software_engineering/`, `todos/`, `work/`, etc. are fully isolated.

## Encoding

All files are written as UTF-8 without BOM, LF line endings. Frontmatter is standard YAML enclosed in `---` delimiters.
