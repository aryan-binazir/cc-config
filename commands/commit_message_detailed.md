---
description: Create detailed commit with comprehensive description
version: "2.0"
---

# Detailed Commit Command

Analyze staged changes and create a comprehensive commit with detailed description:

## Process:

1. **Validate staged changes:**
   - Run `git diff --cached --name-only` to check for staged files
   - If no staged changes, exit with message "No staged changes to commit"

2. **Check project context:**
   - Look at recent commits with `git log --oneline -10` to understand commit message patterns
   - Check if project follows conventional commits or other standards

3. **Analyze changes:**
   - Run `git diff --cached --stat` for file summary
   - Run `git diff --cached` for detailed changes
   - Categorize changes by type: features, fixes, refactoring, docs, tests, etc.

4. **Generate commit:**
   - Auto-detect branch: `git branch --show-current`
   - Create concise title: `[branch-name] - [primary change description]`
   - Generate description with overview, file breakdown, impact, and technical notes

5. **Execute commit:**
   - Run `git commit -m "<title>" -m "<detailed description>"`

## Description Format:
```
## Overview
Brief explanation of what was changed and why

## Changes
- component/file: Description of change

## Impact
- Key benefits and effects
```

## Guidelines:
- Keep title under 72 characters
- Use imperative mood for title
- Be thorough but relevant in description
- Focus on business value and technical impact

Usage: Run without arguments (auto-detects current branch)
