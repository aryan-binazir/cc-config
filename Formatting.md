# CLAUDE.md Formatting Guidelines

## Best Practices

**Be specific**: Use precise instructions like "Use 2-space indentation" rather than vague ones

**Organize with markdown headings and bullet points** for clarity

**Periodically review and update** your memories to keep them relevant

## Formatting Guidelines

- Use standard markdown formatting with `#` headers and bullet points
- You can import other files using `@path/to/import` syntax
- Structure content logically with clear sections

## Quick Management

- Use `#` command to quickly add memories
- Use `/memory` command to edit memory files
- Use `/init` to bootstrap a new CLAUDE.md file

## Types of CLAUDE.md Files

1. **Project memory** (`./CLAUDE.md`): Team-shared instructions
2. **User memory** (`~/.claude/CLAUDE.md`): Personal preferences for all projects
3. **Deprecated local project memory** (`./CLAUDE.local.md`)

## Key Features

- Automatically loaded when Claude launches
- Can import other files using `@path/to/import` syntax
- Discovered recursively in current and parent directories