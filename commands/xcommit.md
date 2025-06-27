---
description: Analyze staged changes and create a commit message
---

Analyze staged changes and create a detailed commit message:

1. Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
2. Run `git diff --cached` to see all staged changes
3. Analyze the changes and generate a concise but descriptive commit message
4. Create commit message in format to use in 5: "[branch] - [detailed description of changes]"
5. Run git commit -m "<insert message from 4>"

Usage: Pass branch name as first argument
