---
description: Format only Git-changed files with project-configured formatters
---

Format only Git-changed files with project-configured formatters:

1. Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
2. Get changed files from Git (staged, unstaged, or branch changes)
3. Check project config files for formatter preferences and custom commands:
   - package.json scripts (format, prettier, format-code)
   - pyproject.toml formatter configuration
   - .prettierrc, .editorconfig settings
   - Cargo.toml, go.mod for language defaults
4. Group changed files by language and apply appropriate formatter:
   - Go: gofmt, goimports
   - JavaScript/TypeScript: prettier
   - Python: pyink (preferred)
   - Rust: rustfmt
   - Java: google-java-format, spotless
5. Respect project-specific formatter configurations and settings

Formatter priority:
1. Custom scripts in package.json/pyproject.toml
2. Project config files (.prettierrc, pyproject.toml)
3. Language defaults (gofmt, rustfmt)

Usage: Run from project root directory
