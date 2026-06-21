---
name: rocket-plan-codex
description: Legacy alias for `rocket-plan standard`. Use this only when the user explicitly says `rocket-plan-codex`; otherwise prefer `rocket-plan` with the configured default or requested profile.
---

# Rocket Plan Codex Legacy Alias

This skill has no standalone workflow.

Immediately use the canonical skill at `skills/rocket-plan/SKILL.md` with plan profile `standard`, as if the user had invoked:

```text
rocket-plan standard
```

Rules:
- Read and follow `skills/rocket-plan/SKILL.md`.
- Resolve profile `standard` from `skills/rocket/rocket.local.yaml` first, then `skills/rocket/rocket.example.yaml`.
- If profile `standard` is missing, stop and report the missing profile.
- Do not reimplement the old `rocket-plan-codex` workflow inline.
