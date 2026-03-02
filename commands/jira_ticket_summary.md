---
description: Generate a PR summary from branch changes
version: "2.0"
---

# Generate PR Summary

Create a short PR summary from the current branch's changes against base.

## Steps:

1. **Detect base branch:**
   - `git symbolic-ref refs/remotes/origin/HEAD | cut -d'/' -f4` (fallback: main → master)

2. **Read the diff:**
   - `git log --oneline <base>..HEAD`
   - `git diff <base>...HEAD`

3. **Write the summary** using this format:

```
## Title
[One-line description of what this PR does]

## Why
[1-2 sentences on the motivation — what problem, request, or improvement drove this]

## What changed
- [Change 1]
- [Change 2]
- [Change 3]
```

## Rules:
- Title is a short imperative phrase (e.g. "Add retry logic to webhook delivery")
- Why section: 1-2 sentences max. Problem or motivation only.
- What changed: tight bullet list. Each bullet is one concrete change. No sub-bullets.
- Skip anything obvious from the title. No filler, no deployment notes, no emoji.
- Omit "What changed" bullets for trivial stuff (whitespace, imports) unless that's the whole PR.