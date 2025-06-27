---
description: Type check only Git-changed files with language-specific tools
---

Type check only Git-changed files with language-specific tools:

1. Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
2. Get changed files from Git (staged, unstaged, or branch changes)
3. Filter files by language and run appropriate type checker:
   - TypeScript: tsc --noEmit on changed .ts/.tsx files
   - Python: mypy on changed .py files
   - Go: go vet on changed .go files
   - Java: javac type checking on changed .java files
   - C#: dotnet build for type checking
4. Check project config files for custom type checking commands:
   - package.json scripts (typecheck, type-check)
   - pyproject.toml mypy configuration
   - tsconfig.json settings
5. Report type errors only for changed files and their dependencies

Supported type checkers:
- TypeScript: tsc, @typescript-eslint
- Python: mypy, pyright, pyre
- Go: go vet, staticcheck
- Java: javac, Error Prone

Usage: Run from project root directory
