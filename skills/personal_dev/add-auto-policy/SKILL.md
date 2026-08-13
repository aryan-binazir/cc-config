---
name: add-auto-policy
description: Add one approve or deny rule to both Claude Code Auto mode and Codex Auto-review, translating the same intent into each native policy format. Use only when explicitly invoked with one action and approve or deny; use align-auto-review for broader comparison, removal, or synchronization.
---

# Add Auto Policy

Mirror intent, not syntax.

1. Require one action plus `approve` or `deny`; stop if either is ambiguous.
2. Read and parse `~/.claude/settings.json` and `~/.codex/config.toml` first.
   Preserve unrelated settings and stricter rules. Skip each policy already
   covered by an equivalent or broader same-strength rule; if both, report
   `already present`.
3. Add the narrowest equivalent prose rule only where missing:
   - Claude: `autoMode.allow` for approve; `autoMode.hard_deny` for deny.
     Start a missing list with `"$defaults"`; preserve an existing list's
     deliberate sentinel choice.
   - Codex: append a local `[auto_review].policy` `Outcome rule: allow ...` or
     `Outcome rule: deny ... regardless of user authorization`. This policy is
     supplemental: append only the new rule, preserving its existing text and
     leaving the built-in policy where it lives.
4. Use Claude `soft_deny` or an overridable Codex denial only when explicitly
   requested. Write only to `autoMode` and `[auto_review].policy`;
   `permissions`, `.rules`, and activation settings stay untouched.
5. Stop on malformed config, conflict, unverifiable defaults, or weaker mapping.
   Validate both prospective JSON/JSONC and TOML documents before writing either,
   roll back both on any write or reread failure, then report exact additions,
   Claude `auto-mode config`, and Codex `approval_policy`/`approvals_reviewer`.
