---
name: obsidian
description: How to interact with Obsidian vaults using the `obsidian` CLI tool. Use this skill whenever the user asks to read, write, search, create, or manage notes in Obsidian, work with daily notes, tags, properties, tasks, or anything involving their Obsidian vault. Also use when the user mentions Obsidian by name, asks about their notes or vault, or wants to append/prepend content to a note. Even if they don't say "Obsidian" explicitly, trigger this if they reference vault notes, daily notes, wikilinks, or frontmatter properties in a personal knowledge base context.
---

# Obsidian CLI

The user has the Obsidian CLI installed at `/opt/homebrew/bin/obsidian`. All interactions with Obsidian vaults go through this tool via `Bash`. Do not try to read vault files directly from the filesystem — always use the CLI, because it routes commands through the running Obsidian app and respects plugins, templates, and sync.

## Quick Start

Before doing anything, discover what vaults exist and find the target file:

```bash
# List vaults (always do this first if you don't know the vault name)
obsidian vaults verbose

# Find a file by searching
obsidian search query="meeting notes" vault="MyVault"

# Browse files in a folder
obsidian files folder="projects" vault="MyVault"
```

## Core Commands

### Reading

```bash
obsidian read path="folder/note.md"
obsidian read file="note name"          # resolves like a wikilink
```

### Writing to existing files

```bash
obsidian append path="folder/note.md" content="New section content"
obsidian prepend path="folder/note.md" content="Top of file content"

# Inline (no newline before content)
obsidian append path="folder/note.md" content=" continued text" inline
```

### Creating new files

```bash
obsidian create name="Weekly Review" content="# Weekly Review\n\n- Item 1"
obsidian create path="projects/new_project.md" content="# New Project"
obsidian create name="From Template" template="Meeting Notes"
```

### Searching

```bash
# Returns file paths only
obsidian search query="quarterly review"

# Returns matching lines with context
obsidian search:context query="action items"

# Scoped to a folder
obsidian search query="budget" path="work"
```

### Daily Notes

```bash
obsidian daily:read                          # read today's daily note
obsidian daily:append content="- 3pm call with team"
obsidian daily:prepend content="## Morning\n\n- Focus block"
obsidian daily:path                          # get the file path
```

### Properties (Frontmatter)

```bash
obsidian property:set name="status" value="active" path="projects/alpha.md"
obsidian property:read name="status" path="projects/alpha.md"
obsidian property:remove name="old_field" path="projects/alpha.md"
obsidian properties path="projects/alpha.md"    # list all properties on a file
```

### Tags

```bash
obsidian tags                                # all tags in vault
obsidian tags path="projects/alpha.md"       # tags on a specific file
obsidian tag name="project" verbose          # files with this tag
```

### Tasks

```bash
obsidian tasks                               # all tasks in vault
obsidian tasks todo                          # incomplete only
obsidian tasks done                          # completed only
obsidian tasks path="projects/alpha.md"      # tasks in a specific file
obsidian task path="projects/alpha.md" line=12 toggle   # toggle a task
obsidian task path="projects/alpha.md" line=12 done     # mark done
```

## Important Patterns

### Vault targeting

When multiple vaults exist, pass `vault="VaultName"` to every command:

```bash
obsidian read path="note.md" vault="Work"
obsidian search query="meeting" vault="Personal"
```

### File vs Path

- `file="note name"` — resolves by name like a wikilink (finds `note name.md` anywhere in the vault)
- `path="folder/note.md"` — exact path from vault root

Use `path=` when you know where the file is. Use `file=` when you only know the name.

### Content formatting

Use `\n` for newlines and `\t` for tabs in content strings:

```bash
obsidian append path="log.md" content="\n## 2026-03-28\n\n- First item\n- Second item"
```

### Quoting

Quote values that contain spaces:

```bash
obsidian read file="my long note name"
obsidian create name="Project Alpha Notes" content="# Notes"
```

## Gotchas

1. **Empty output on search is not an error.** It just means no matches. Try broader terms, or use `obsidian files` to browse and find the right name.
2. **Obsidian must be running.** The CLI talks to the app. If commands silently fail, the app may not be open.
3. **Don't read vault files directly from disk.** Always use the CLI. The filesystem path can be found via `obsidian vault info=path` if needed, but prefer CLI commands for reads/writes.

## Less Common but Useful

```bash
obsidian backlinks file="note name"          # what links to this note
obsidian links path="folder/note.md"         # outgoing links from a note
obsidian outline path="folder/note.md"       # heading structure
obsidian move path="old/note.md" to="new/"   # move/rename
obsidian delete path="folder/note.md"        # trash a file
obsidian folders                             # list all folders
obsidian templates                           # list available templates
```
