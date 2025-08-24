---
description: Create detailed commit with comprehensive description
version: "1.1"
---

# Detailed Commit Command

Analyze staged changes and create a comprehensive commit with detailed description:

## Process:

1. **Validate staged changes:**
   - Run `git diff --cached --name-only` to check for staged files
   - If no staged changes, exit with message "No staged changes to commit"
   
2. **Check project context:**
   - Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
   - Look at recent commits with `git log --oneline -10` to understand commit message patterns
   - Check if project follows conventional commits or other standards
   
3. **Analyze changes thoroughly:**
   - Run `git diff --cached --stat` for file summary
   - Run `git diff --cached` for detailed changes
   - Categorize changes by type: features, fixes, refactoring, docs, tests, etc.
   
4. **Generate commit:**
   - Auto-detect branch: `git branch --show-current`
   - Create concise title: `[branch-name] - [primary change description]`
   - Generate comprehensive description including:
     - **Overview**: What was changed and why
     - **File breakdown**: Modifications by component/file
     - **Impact**: Benefits and effects of changes
     - **Technical notes**: Important implementation details
     - **Testing**: Any test changes or requirements
   
5. **Execute commit:**
   - Run `git commit -m "<title>" -m "<detailed description>"`

## Description Format:
```
## Overview
Brief explanation of what was changed and motivation

## Changes
- file1.js: Added new feature X
- file2.py: Fixed bug in Y function  
- tests/: Updated test cases for new functionality

## Impact
- Improves performance by X%
- Fixes issue #123
- Enables future feature Y

## Technical Details
- Uses new algorithm Z for better efficiency
- Refactored legacy code for maintainability
```

## Guidelines:
- Keep title under 72 characters
- Use imperative mood for title
- Be thorough but relevant in description
- Focus on business value and technical impact

Usage: Run without arguments (auto-detects current branch)