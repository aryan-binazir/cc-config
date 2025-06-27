---
description: Create detailed commit with comprehensive description
---

Analyze staged changes and create a detailed commit message with comprehensive description:

1. Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
2. Run `git diff --cached` to see all staged changes
3. Analyze the changes and generate a concise but descriptive commit message
4. Create commit message in format to use in 6: "[branch] - [detailed description of changes]"
5. Generate a comprehensive description of the changes in Markdown format, including:
   - Overview of what was changed and why
   - Detailed breakdown of modifications by file/component
   - Impact and benefits of the changes
   - Any technical details worth noting
   - DO NOT include Claude co-authorship or other unnecessary details
6. Run git commit -m "<insert message from 4>" -m "<insert comprehensive description from 5>"

Usage: Pass branch name as first argument