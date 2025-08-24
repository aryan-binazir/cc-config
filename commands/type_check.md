---
description: Type check only Git-changed files with language-specific tools
version: "1.1"
---

# Type Check Changed Files Command

Run type checking only on files changed in Git with appropriate language-specific type checkers.

## Process:

1. **Check project configuration:**
   - Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
   - Look for project type checker configs: `tsconfig.json`, `pyproject.toml`, `go.mod`, etc.
   - Check package.json for custom typecheck scripts

2. **Get changed files:**
   - Staged files: `git diff --cached --name-only`
   - Unstaged files: `git diff --name-only`
   - Branch changes: `git diff origin/main..HEAD --name-only` (with fallback)
   - Filter out deleted files: `git diff --name-only --diff-filter=d`

3. **Group files by language and run type checkers:**

## Language Support & Tools:

| Language | Primary Tool | Command | Fallback |
|----------|--------------|---------|----------|
| TypeScript/JavaScript | tsc | `tsc --noEmit` on changed files | eslint @typescript-eslint |
| Python | mypy | `mypy <changed-files>` | pyright, then pyre |
| Go | go vet | `go vet <changed-packages>` | staticcheck |
| Java | javac | `javac -Xlint <changed-files>` | Error Prone |
| C# | dotnet | `dotnet build --no-restore` | - |
| Rust | cargo check | `cargo check` (workspace-aware) | - |

4. **Handle project-specific configurations:**
   - **TypeScript**: Use project's `tsconfig.json`, respect `exclude` patterns
   - **Python**: Honor `pyproject.toml` mypy config, virtual env detection
   - **Go**: Check Go modules, run on affected packages only
   - **Java**: Respect classpath and project structure

5. **Smart error reporting:**
   - Report type errors only for changed files and their direct dependencies
   - Show file-relative paths for easier navigation
   - Separate errors by language for clarity
   - Exit with non-zero code if type errors found

## Error Handling:
- Skip missing type checker tools with informative warnings
- Suggest installation commands for missing tools
- Continue checking other languages if one fails
- Handle projects with multiple language configurations

## Performance Optimizations:
- Run type checkers only on relevant file subsets
- Use incremental checking when supported (tsc --incremental)
- Parallel execution for independent language checks
- Cache type checker results when possible

Usage: Run from project root directory. Automatically detects and processes all changed files by language.
