---
description: Lint only Git-changed files with language-specific linters and auto-fix
version: "2.0"
---

# Lint Changed Files Command

Lint only files changed in Git with appropriate language-specific linters and auto-fix capabilities.

## Process:

1. **Check project configuration:**
   - Look for project linter configs: `.eslintrc*`, `pyproject.toml`, `cargo.toml`, etc.
   - Check package.json for custom lint scripts

2. **Get changed files:**
   - Staged files: `git diff --cached --name-only`
   - Unstaged files: `git diff --name-only`
   - Branch changes: `git diff origin/main..HEAD --name-only` (with fallback)
   - Filter out deleted files: `git diff --name-only --diff-filter=d`

3. **Group and lint by language:**
   - **JavaScript/TypeScript**: `eslint --fix` or npm script, then `prettier --write`
   - **Python**: `ruff check --fix`, fallback to `flake8` (read-only)
   - **Go**: `gofmt -w`, then `golangci-lint run --fix`
   - **Rust**: `cargo clippy --fix --allow-dirty`
   - **Java**: `google-java-format --replace`
   - **Other**: Check for project-specific formatters

4. **Report results:**
   - Show files processed and fixes applied
   - Report any remaining issues that require manual attention
   - Exit with non-zero code if unfixable issues remain

## Supported Languages & Tools:

| Language | Linter | Auto-fix | Fallback |
|----------|---------|----------|----------|
| JS/TS | ESLint + Prettier | Yes | Standard |
| Python | Ruff | Yes | Flake8 (read-only) |
| Go | golangci-lint + gofmt | Yes | go vet |
| Rust | Clippy | Yes | rustfmt |
| Java | google-java-format | Yes | checkstyle |

## Error Handling:
- Skip missing linter tools with warning
- Continue processing other languages if one fails
- Provide installation suggestions for missing tools

Usage: Run from project root directory. Processes all changed files automatically.
