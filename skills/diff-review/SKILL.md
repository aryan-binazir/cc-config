---
name: diff-review
description: Review uncommitted changes against the original user ask, ask clarifying questions before the full review when needed, and save the resulting review to `_scratch/reviews/`. Use when the user wants a brutal review of current uncommitted work against an explicit requirement or scope.
---

# Diff Review

Review the current uncommitted changes against the original requirement, not just against general code quality.

## Required Input

You need a plain-language description of the original ask. If the user has not supplied it, ask for it and stop there.

## Constraints

This is a read-and-review workflow. Do not modify any source files. Only read code, run git commands for context, and write the final review file.

## Workflow

1. Read the user's description of the original ask.
2. Gather context by running these commands:
   - `git branch --show-current`
   - `git status --short`
   - `git diff HEAD`
   These are the only git commands you need.
3. Before writing the full review, ask any clarifying questions that are necessary to judge scope, intent, or tradeoffs.
4. After the user answers, review the uncommitted changes as a brutal code reviewer. Treat the changes as if another agent attempted to fulfill the original requirement. Flag specifically:
   - overengineering
   - unnecessary code
   - scope creep
   - bugs
   - anything that does not directly serve the original ask
5. Save the review to `_scratch/reviews/{branch}-review.md`.

## Output

Be specific and use file and line references where possible. Findings should be concise, blunt, and grounded in the diff.
