---
description: Walk through a PR bottom-up, explaining each change and how it connects
version: "1.1"
---

# PR Walkthrough

Walk me through this PR starting from the base of the changes. Help me understand each step — what was added or changed, why it matters, and how the pieces connect.

This is a guided tour, not a code review. No suggestions. No judgments. Just clear explanation.

Explain everything in plain language as if the reader is not familiar with this codebase. Avoid jargon or internal shorthand — if a concept needs context, give it. Name things by what they do, not just what they're called.

## Get Changes

```bash
# Current branch
BRANCH=$(git branch --show-current)

# Find the merge-base (where this branch diverged from main)
BASE=$(git merge-base origin/main HEAD 2>/dev/null || git merge-base origin/master HEAD)

# List commits on this branch (oldest first)
git log --oneline --no-merges --reverse $BASE..HEAD

# Full diff
git diff $BASE..HEAD

# Files changed summary
git diff --stat $BASE..HEAD
```

## Build the Walkthrough

Start from the lowest-level changes (types, utilities, things nothing else depends on) and work up to the entry points (routes, handlers, CLI commands — where behavior is triggered).

For each change:
- Say what was added or modified, and why it matters.
- Show me where in the code.
- Connect it to the other changes in this PR — what calls it, what it depends on.

After walking through the individual changes, trace 1-3 key paths end-to-end so I can see how the PR works as a whole.

Keep it concrete. A couple sentences per item max.
