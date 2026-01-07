---
description: Analyze staged changes and create a commit message
version: "3.0"
---

# Commit Message

Analyze staged changes and generate a commit message.

## Process

1. Validate: `git diff --cached --name-only` — exit if nothing staged
2. Branch: `git branch --show-current` — extract ticket ID (strip leading initials like `ab-`)
3. Context: `git log --oneline -5` — match project's commit style
4. Analyze: `git diff --cached` — understand what changed and why

## Output

```
[TICKET-ID - title: imperative, <72 chars, what changed]

[body: why it changed, key files affected, any breaking changes]
```

Body is optional for trivial changes. Include it when:
- Multiple files/components affected
- Non-obvious reasoning
- Breaking changes or migration notes

## Guidelines
- Imperative mood ("Add", "Fix", "Update")
- Focus on why, not how (code shows how)
- Be specific but not verbose
