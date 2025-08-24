---
description: Lint only Git-changed files with language-specific linters and auto-fix
version: "1.1"
---

# Lint Changed Files Command

Lint only files changed in Git with appropriate language-specific linters and auto-fix capabilities.

## Process:

1. **Check project configuration:**
   - Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
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

4. **Handle project-specific tools:**
   - Check for custom lint commands in package.json scripts
   - Use project's preferred linter configuration
   - Respect .gitignore and linter ignore files

5. **Report results:**
   - Show files processed and fixes applied
   - Report any remaining issues that require manual attention
   - Exit with non-zero code if unfixable issues remain

## Supported Languages & Tools:

| Language | Linter | Auto-fix | Fallback |
|----------|---------|----------|----------|
| JS/TS | ESLint + Prettier | ✅ | Standard |
| Python | Ruff | ✅ | Flake8 (read-only) |
| Go | golangci-lint + gofmt | ✅ | go vet |  
| Rust | Clippy | ✅ | rustfmt |
| Java | google-java-format | ✅ | checkstyle |

## Error Handling:
- Skip missing linter tools with warning
- Continue processing other languages if one fails
- Provide installation suggestions for missing tools

Usage: Run from project root directory. Processes all changed files automatically.