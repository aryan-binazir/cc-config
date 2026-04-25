# Database representation

Notion databases are the trickiest object to represent in Obsidian. A Notion database combines two things that Obsidian has separate concepts for:

1. A **collection of pages** with shared schema (rows)
2. A **view** over those pages (table, board, gallery, calendar)

Obsidian doesn't have a native equivalent. The closest analog is the Dataview plugin, which can query frontmatter across notes. This skill represents databases in a way that makes Dataview (or just vanilla search) work well.

## The representation

Each database becomes:

- A **folder** named after the database title
- An `_index.md` inside that folder, showing a markdown table view of the rows
- One `.md` file per row, with all database properties flattened into frontmatter

```
Notion/Q2 planning/
├── _index.md                   ← database table + description
├── Ship docs.md                ← row
├── Talk to legal.md            ← row
└── Review pricing model.md     ← row
```

## Why this shape

- **Each row is a real note**, so Obsidian search / backlinks / Dataview / templates / the graph view all see it. This is the main win over a JSON or CSV dump.
- **Frontmatter carries the schema**, so `LIST FROM "Notion/Q2 planning" WHERE Status = "In Progress"` just works in Dataview.
- **The folder is the database**, so the user can move or rename the folder and the shape stays intact.
- **The `_index.md` is a quick-glance table**, so you don't need to query Dataview to see what's in the database.

## `_index.md` content

```markdown
---
notion_id: <database_id>
notion_url: https://www.notion.so/...
notion_type: database
last_notion_edited_time: 2026-04-22T15:00:00Z
last_synced_at: 2026-04-23T10:00:12Z
---

# Q2 planning

<database description block, if Notion has one>

## Rows (sorted by last edited, descending)

| Title | Status | Owner | Priority | Due date |
|---|---|---|---|---|
| [[Ship docs]] | In Progress | Ar | High | 2026-05-01 |
| [[Talk to legal]] | Not started | Ar | Medium | 2026-05-10 |
| [[Review pricing model]] | Done | Co-founder | High | 2026-04-15 |
```

The columns are the database's visible properties (in Notion's property order). Rows are sorted by `last_edited_time` descending by default.

Wikilinks in the table are bare-filename (`[[Ship docs]]`) rather than full-path (`[[Notion/Q2 planning/Ship docs]]`) because Obsidian resolves wikilinks by filename across the whole vault. Bare names are cleaner and survive moves.

If the vault has a filename collision (two rows titled "Untitled" from different databases), fall back to path-qualified wikilinks for the colliding entries: `[[Notion/Q2 planning/Untitled|Untitled]]`.

## Row file frontmatter

Every database row gets its Notion properties flattened into YAML frontmatter:

```markdown
---
notion_id: <page_id>
notion_url: https://www.notion.so/...
last_notion_edited_time: 2026-04-22T15:00:00Z
last_synced_at: 2026-04-23T10:00:12Z
db_parent: <database_id>
# --- Notion properties below ---
Status: "In Progress"
Priority: "High"
Owner: "Ar"
"Due date": 2026-05-01
Tags:
  - work
  - urgent
---

# Ship docs

<the row page's block content, converted via the usual rules>
```

## Property type mapping

| Notion property type | YAML representation |
|---|---|
| `title` | used as the filename; also reproduced as the H1 heading in the body |
| `rich_text` | string (plain text rendering, no formatting — formatting stays in the body) |
| `number` | number |
| `select` | string |
| `status` | string |
| `multi_select` | array of strings |
| `date` (no end) | ISO date string `"2026-05-01"` or `"2026-05-01T14:00:00Z"` if time is set |
| `date` (with end) | object: `{start: "...", end: "..."}` |
| `people` | array of names (string) |
| `files` | array of filenames; files downloaded to attachments, links appear in body not frontmatter |
| `checkbox` | bool |
| `url` | string |
| `email` | string |
| `phone_number` | string |
| `formula` | resolved value using its underlying type's mapping |
| `relation` | array of row titles as wikilink strings like `"[[Other Row]]"` (stored as string so YAML doesn't choke on wikilinks — user can still read them; Dataview treats them as links) |
| `rollup` | resolved value using underlying type's mapping |
| `created_time` | ISO datetime |
| `created_by` | string (user name) |
| `last_edited_time` | ISO datetime |
| `last_edited_by` | string (user name) |

### Property name handling in YAML

Property names with spaces, punctuation, or leading uppercase need quoting in YAML. Quote conservatively (always quote keys with spaces, slashes, colons, or dots) to avoid surprises.

### Relation properties

Relations point to pages in other databases. They're serialized as an array of wikilink-formatted strings in frontmatter:

```yaml
Related projects:
  - "[[Acme rebrand]]"
  - "[[Q2 planning]]"
```

This keeps the link visible in frontmatter. In the body, the same relation is also rendered as a bulleted list of wikilinks under an "Relations" section — so the user can navigate from either place.

## Database of databases

Rare but possible: a database whose rows are themselves databases. Recurse. The outer database folder contains `_index.md` + one subfolder per row, where each row subfolder is itself a database folder.

```
Notion/Outer DB/
├── _index.md
├── Inner DB A/
│   ├── _index.md
│   └── row_1.md
└── Inner DB B/
    ├── _index.md
    └── row_1.md
```

## Empty databases

An empty database still gets an `_index.md` with the headers and an empty table body:

```markdown
## Rows

_No rows yet._
```

## Database views

Notion lets users define multiple views (table, board, gallery, etc.). This skill only mirrors the underlying data, not the views. The `_index.md` is always a table regardless of what view the user has in Notion. This is an intentional simplification — representing every view type would be a large effort with marginal benefit for an Obsidian-based workflow.

## Regenerating `_index.md` when rows change

When any row in a database is created, updated, or renamed (but the database itself wasn't otherwise edited), the `_index.md` table becomes stale. Regenerate the `_index.md` at the end of processing each database's rows. Track this with a set of "dirty databases" during the run — add to it whenever a row is written, and regenerate all dirty database indices in a final pass.

Regenerating `_index.md` still goes through the conflict detection — if the user has edited the table in Obsidian, preserve their edit to a sidecar before overwriting.
