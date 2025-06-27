Lint only files changed in Git with appropriate language-specific linters and auto-fix:

1. Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
2. Get changed files from Git (staged, unstaged, or branch changes)
3. Group files by language extension
4. Run appropriate linter with --fix flag on changed files only
5. Report any remaining issues

Languages supported:
- Go: gopls check . (uses Go language server)
- Python: mypy . (static type checking)
- TypeScript/JavaScript: Check package.json for scripts like npm run lint, npm run type-check, or npm run build
- Rust: cargo clippy --fix --allow-dirty

Usage: Run from project root directory