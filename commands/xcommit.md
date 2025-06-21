Analyze staged changes and create a detailed commit message:

1. Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
2. Run `git diff --cached` to see all staged changes
3. Analyze the changes and generate a concise but descriptive commit message
4. Output the commit message in format: "[branch] - [detailed description of changes]"

Usage: Pass branch name as first argument