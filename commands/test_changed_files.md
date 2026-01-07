---
description: Run tests for Git-changed files and their associated test files
version: "1.0"
---

# Test Changed Files Command

Run tests only for files changed in Git, including their associated test files.

## Process:

1. **Check project configuration:**
   - Look for test configs: `jest.config.js`, `pytest.ini`, `go.mod`, `cargo.toml`
   - Check package.json for custom test scripts

2. **Get changed files:**
   - Staged files: `git diff --cached --name-only`
   - Unstaged files: `git diff --name-only`
   - Branch changes: `git diff origin/main..HEAD --name-only` (with fallback)
   - Filter out deleted files: `git diff --name-only --diff-filter=d`

3. **Find associated test files:**
   - For each changed file, detect corresponding test files
   - Patterns: `*.test.js`, `*_test.py`, `*_test.go`, `test_*.py`, etc.
   - Include test files that import/reference changed files

4. **Run tests by language:**

| Language | Test Runner | Command | Fallback |
|----------|-------------|---------|----------|
| JS/TS | Jest | `jest <test-files>` | npm test |
| Python | pytest | `pytest <test-files>` | unittest |
| Go | go test | `go test <packages>` | - |
| Rust | cargo test | `cargo test` | - |
| Java | JUnit | `mvn test` or `gradle test` | - |

5. **Report results:**
   - Show test files executed and pass/fail counts
   - Display failed test output with file context
   - Exit with non-zero code if any tests fail

## Error Handling:
- Skip languages with missing test tools (show warning)
- Continue testing other languages if one fails
- Suggest installation commands for missing tools

Usage: Run from project root. Processes only changed files and their tests.
