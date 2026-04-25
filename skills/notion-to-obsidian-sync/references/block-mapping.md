# Notion block → Obsidian markdown mapping

Authoritative reference. The converter (`scripts/convert_page.py`) follows this table. If a block type isn't listed here, the converter emits a warning callout wrapping the raw Notion JSON, so content is never silently dropped.

## Conventions

- "Rich text" = Notion's inline text array with annotations. See the [Inline rich text](#inline-rich-text) section.
- "Children" = blocks nested inside another block (lists, toggles, columns, etc.). Recurse with increased indent level.
- Indent uses 4 spaces per level (Obsidian's default for nested lists).

## Text blocks

| Notion block | Obsidian |
|---|---|
| `paragraph` | plain rich text, with a blank line before and after |
| `heading_1` | `# <rich text>` |
| `heading_2` | `## <rich text>` |
| `heading_3` | `### <rich text>` |
| `quote` | each line prefixed with `> ` |
| `callout` | Obsidian callout — see [Callouts](#callouts) |
| `toggle` | see [Toggles](#toggles) |
| `divider` | `---` on its own line |
| `code` | fenced code block with language. Caption (if any) as italic line below. See [Code](#code) |
| `equation` (block-level) | `$$<expression>$$` on its own line |

### Heading-toggle variants

A heading block with `is_toggleable: true` has children. Render as the heading plus children directly after — Obsidian's folding is heading-based, so the "toggle" behavior is preserved implicitly (click the heading to fold).

## Code

```
```<language>
<content>
```
```

The language comes from Notion's `code.language`. Obsidian recognizes most Notion languages as-is. Special cases:

| Notion language | Obsidian language tag |
|---|---|
| `plain text` | (no tag) |
| `abap` / `c#` | `abap` / `csharp` |
| `mermaid` | `mermaid` (Obsidian renders this natively if Mermaid is enabled) |

Captions (Notion `code.caption`) render as `_<caption>_` on a line immediately below the fence.

## Callouts

Notion callouts have an icon (emoji or file) and a colored background. Map to Obsidian callouts:

| Emoji in Notion icon | Obsidian callout type |
|---|---|
| ℹ️ / 💡 | `[!info]` |
| ⚠️ / ⚠ | `[!warning]` |
| 🚨 / 🛑 / ❗ | `[!danger]` |
| ✅ / 👍 | `[!success]` |
| ❌ / 👎 | `[!failure]` |
| 📝 / 📌 / 🗒️ | `[!note]` |
| ❓ / ❔ | `[!question]` |
| 💬 | `[!quote]` |
| 📋 | `[!todo]` |
| 🎯 / ⭐ | `[!tip]` |
| (other or file icon) | `[!note]` with the original icon emoji prepended to the first line of the body |

Format:

```markdown
> [!note] <first line of rich text, if any>
> <rest of rich text>
> <children, also `> ` prefixed and indented appropriately>
```

If the callout has no icon or a file icon, use `[!note]`.

## Toggles

Notion toggles are collapsible. Obsidian has two options:

1. `<details><summary>...</summary>...</details>` — works but renders as plain HTML, not native
2. Heading-based folding (only works if the toggle contains heading-like content)

Default: use `<details>`:

```markdown
<details>
<summary><toggle title rich text></summary>

<children blocks, rendered normally>

</details>
```

The blank lines around the inner content matter — without them, Obsidian won't render the markdown inside the `<details>`.

## List blocks

| Notion block | Obsidian |
|---|---|
| `bulleted_list_item` | `- <rich text>` |
| `numbered_list_item` | `1. <rich text>` (Obsidian auto-renumbers on render) |
| `to_do` | `- [ ] <rich text>` or `- [x] <rich text>` based on `to_do.checked` |

Nested items indent with 4 spaces per level.

## Tables

```markdown
| col 1 | col 2 | col 3 |
|---|---|---|
| a | b | c |
| d | e | f |
```

- First row is the header if Notion's `table.has_column_header` is `true`; otherwise insert a blank header row (markdown requires one).
- Cell contents are the cell's rich text, with `\n` replaced by `<br>` (markdown tables don't support newlines).
- Pipe characters in cells escaped as `\|`.

## Media blocks

| Notion block | Obsidian |
|---|---|
| `image` (Notion-hosted file) | download → `![[Notion/_attachments/<page_id>/<file>]]` |
| `image` (external URL) | `![<caption>](<url>)` |
| `video` / `audio` / `pdf` / `file` (Notion-hosted) | download → `![[Notion/_attachments/<page_id>/<file>]]` |
| `video` / `audio` / `pdf` / `file` (external URL) | `[<caption or filename>](<url>)` |
| `bookmark` | `[<title or domain>](<url>)` + italic caption if present |
| `embed` | `[<url>](<url>)` with HTML comment `<!-- Notion embed -->` above |
| `link_preview` | same as `embed` |

Captions appear as an italic line immediately below the embed/link.

### Why download Notion-hosted files

Notion's signed-URL scheme for uploaded files expires (typically after an hour). If we left the URLs in place, the markdown would break silently after the links expire. Downloading to `_attachments/` makes the vault self-contained.

## Inline rich text

Notion rich text is an array of objects, each with `plain_text`, `annotations`, and optionally a `href` and type-specific fields. Combine annotations in the order shown (from outermost to innermost wrapper):

| Annotation | Obsidian |
|---|---|
| `bold` | `**text**` |
| `italic` | `*text*` |
| `strikethrough` | `~~text~~` |
| `code` (inline) | `` `text` `` |
| `underline` | `<u>text</u>` (no markdown for underline; Obsidian renders the HTML) |
| `color` | dropped (Obsidian has no inline color; future: Obsidian CSS classes) |

Combined example: `**_bold italic_**` for text that is both bold and italic.

Inline equations: `{type: "equation", equation: {expression: "E=mc^2"}}` → `$E=mc^2$`.

Links: if `href` is set, wrap the result in `[text](url)`.

## Mentions (inline)

| Mention type | Obsidian |
|---|---|
| `user` | `<user name>` (plain text — Obsidian has no user system) |
| `page` | `[[<wikilink>]]` if the target page is in the manifest (synced); else `[<page title>](<notion_url>)` |
| `database` | `[[<db_folder>/_index]]` if synced; else link to Notion |
| `date` (no end) | the ISO date as plain text |
| `date` (with end) | `<start> → <end>` |
| `link_mention` / `link_preview` | `[<title or url>](<url>)` |
| `template_mention` | drop (these are template placeholders, rarely meaningful after sync) |

When resolving page/database mentions to wikilinks, use the bare filename form (`[[Ship docs]]`) unless there's a known collision in the vault, in which case use path-qualified form (`[[Notion/Q2 planning/Ship docs|Ship docs]]`).

## Structural / layout blocks

### Columns

Notion `column_list` contains `column` children side-by-side. Markdown has no column concept. Flatten to sequential content:

```markdown
<!-- Notion columns flattened (N columns in original) -->
<content of column 1>

<content of column 2>

...
```

The HTML comment documents the lossy conversion.

### Synced blocks

Notion synced blocks have an "original" (the source) and "duplicates" (references). Notion doesn't always surface which is which; treat both as real content and render inline. Prefix with:

```markdown
<!-- Notion synced block (id: <uuid>) -->
```

This way, if the user sees the same content twice in two different pages, the comment explains why.

### Breadcrumb / Table of contents

Skip silently. Obsidian has its own ways to do these (breadcrumbs: via Dataview or plugin; TOC: via plugins), and Notion's inline versions don't translate usefully.

### `link_to_page`

A Notion block that's just a reference to another page. Render as `[[<target>]]` wikilink if the target is synced; otherwise a regular link to the Notion URL.

### `child_page`

Inline reference to a child page (appears in the parent page's block stream). Render as `[[<child title>]]`. The child page itself is synced as its own file per the folder layout rules.

### `child_database`

Inline reference to a database that lives inside this page. Render as `[[<db name>/_index]]`. The database is synced as its own folder.

## Unhandled block types

For any type not listed above, wrap in a warning callout so the user sees it and the skill can be extended:

```markdown
> [!warning] Unhandled Notion block type
> Type: `<block_type>`
> ```json
> <pretty-printed block JSON>
> ```
```

Additionally, the converter increments `unhandled_block_types[<block_type>]` in the manifest so the next sync report can surface counts across the whole workspace. This makes it trivial to spot which block types need attention next.

## Frontmatter on every page

All converted pages start with YAML frontmatter. Minimum keys:

```yaml
---
notion_id: <uuid>
notion_url: https://www.notion.so/...
last_notion_edited_time: 2026-04-22T15:00:00Z
last_synced_at: 2026-04-23T10:00:12Z
---
```

Additional keys when applicable:

- `notion_parent_id`: parent page or database UUID
- `notion_type: database` for database `_index.md` files
- `db_parent: <database_id>` for database row files
- Database row properties, flattened (see `databases.md`)

## Block order

Notion blocks are ordered in the page. Preserve order exactly. Children of a block come immediately after that block in the output.

## Whitespace

- One blank line between top-level blocks.
- No blank line between list items at the same level.
- One blank line before and after a fenced code block, table, or callout.
- Trim trailing whitespace from each line.
- File ends with a single trailing newline.
