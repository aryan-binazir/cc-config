---
description: Analyze staged changes and create a commit message
version: "2.0"
---

# Quick Commit Command

Analyze staged changes and create a concise commit message:

## Process:

1. **Validate staged changes:**
   - Run `git diff --cached --name-only` to check for staged files
   - If no staged changes, exit with message "No staged changes to commit"

2. **Check project context:**
   - Look at recent commits with `git log --oneline -5` to understand commit message patterns

3. **Analyze and commit:**
   - Run `git diff --cached` to see all staged changes
   - Auto-detect branch: `git branch --show-current`
   - Use format: `[branch-name] - [concise description]`
   - Run `git commit -m "<generated message>"`

## Guidelines:
- Keep message under 72 characters for the first line
- Focus on what changed, not how
- Use imperative mood ("Add", "Fix", "Update")
- Be specific but concise

Usage: Run without arguments (auto-detects current branch)
