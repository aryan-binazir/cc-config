---
name: align-auto-review
description: Reconcile user-authored Auto-review intent across Claude Code, Codex, and Cursor while preserving native policy models and defaults. Use when asked to add, remove, compare, or synchronize Auto-review policy.
---

# Align Auto-review

Mirror intent, not syntax.

## Surfaces

- Claude: `~/.claude/settings.json` → `autoMode`; also inspect
  `permissions.ask` and `permissions.deny`.
- Codex: `~/.codex/config.toml` → `[auto_review].policy`; inspect root
  `approvals_reviewer` and `approval_policy`. Ignore `.rules` files.
- Cursor activation: `~/.cursor/cli-config.json` → `approvalMode`; policy:
  `~/.cursor/permissions.json` → `autoRun.allow_instructions` and
  `autoRun.block_instructions`. Ignore deterministic Cursor CLI
  `permissions.allow` and `permissions.deny`.

Activation and policy are separate. Report activation state; change it only
when the user asks.

## Reconcile

1. Read and parse all three files before editing. Treat a missing file or
   section as "no local override" — the feature may still be active. Stop on
   malformed input.
2. Reduce user-authored entries to intent plus strength: allow, advisory block,
   mandatory prompt, or hard deny. Exclude vendor defaults.
3. Compare semantically and classify every cross-harness mapping as `exact`,
   `approximate`, or `no native equivalent`.
4. Add missing intent in native form. Approximation is acceptable only when it
   preserves strength: map hard denies to hard-deny primitives and mandatory
   prompts to mandatory-prompt primitives.
5. Stop only for conflict, ambiguous intent, or weakening. Otherwise edit.

## Preserve Defaults

- Claude: retain `"$defaults"` in every customized Auto-mode list; add it when
  creating one.
- Codex: local policy replaces its default. Before the first override, obtain
  the current official default policy, copy it intact into `policy`, then add a
  clearly delimited local-rules section — the default portion stays verbatim.
  If the exact default cannot be verified, stop.
- Cursor: `autoRun` augments built-in review behavior; preserve unrelated keys.

## Finish

Validate JSON/JSONC and TOML, reread all three policies, and compare again. If
aligned, say `already aligned`. Otherwise report only:

- each intent and mapping grade
- exact edits by harness
- unmappable restrictions
- activation state
