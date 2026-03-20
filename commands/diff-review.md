---
description: Review uncommitted branch changes against the original ask and save the review to _scratch/reviews
allowed-tools: Read, Write, Edit, Bash(git branch --show-current), Bash(git status), Bash(git status --short), Bash(git diff HEAD), Bash(mkdir -p _scratch/reviews)
argument-hint: "<plain-text description of the original ask>"
---

$ARGUMENTS

If `$ARGUMENTS` is empty, immediately ask me for a plain-text description of the original ask and do nothing else.

Another agent attempted to fulfill this requirement: $ARGUMENTS. Review all uncommitted changes (staged and unstaged) on the current branch. Be a brutal code reviewer. Flag overengineering, unnecessary code, scope creep, bugs, and anything that doesn't directly serve the original ask. Be specific with file and line references. Before writing the full review, immediately ask me any clarifying questions you have. Do not proceed with the detailed review until I've answered.

Use this live git context:

## Current branch
!`git branch --show-current`

## Git status
!`git status --short`

## Full diff against HEAD
!`git diff HEAD`

After the clarifying questions are answered and only then, write the full review to `_scratch/reviews/!`git branch --show-current`-review.md`.
