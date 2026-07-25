---
name: ask-adversary-in-block
description: Use when the user invokes ask-adversary-in-block or asks to replace the normal human clarification pass with one configured adversarial model call. Collect every question into one block, take the critique seriously, make the final decisions, and report them to the user.
---

# Ask Adversary In Block

Resolve this skill's directory and run:

```bash
uv run --script "<skill-dir>/scripts/resolve_config.py"
```

Stop on failure. Read the `call-claude`, `call-codex`, or `call-cursor` skill
matching the resolved runner and use its wrapper. Always pass the resolved
overrides: `--model` and `--timeout-ms`, plus `--effort` for Claude or
`--reasoning-effort` for Codex. Cursor encodes effort in its model ID. Never
substitute another runner.

Before asking the user ordinary clarification questions, collect every question,
uncertainty, assumption, edge case, and objection into one self-contained prompt.
Make one adversary call containing the original task, relevant evidence and
constraints, and the complete question block. Ask it to answer every question,
find important questions you missed, attack weak assumptions, and give brutally
honest recommendations. Make exactly one adversary call for this invocation. If
it fails, ask the user the collected question block directly.

Treat the response as serious advice, not authority. Decide what to adopt, report
the adversary used and the decisions made, then continue the original task.
Still ask the user when a genuine blocker remains or a decision is hard to undo
or changes user-facing behavior or scope.
