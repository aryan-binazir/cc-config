---
name: ask-adversary-in-block
description: Replace the human clarification pass with one configured adversarial model call — every question in one block — then decide and report. Use when the user invokes ask-adversary-in-block or asks for that swap.
disable-model-invocation: true
---

Configuration loads `ask-adversary-in-block.local.yaml` when present, otherwise `ask-adversary-in-block.example.yaml`.

# Ask Adversary In Block

Resolve this skill's directory and run:

```bash
uv run --script "<skill-dir>/scripts/resolve_config.py"
```

Stop on failure. Read the resolved `call_skill` and use its wrapper only, always passing the resolved `--model` and `--timeout-ms`, plus `--effort` for Claude or `--reasoning-effort` for Codex (Cursor encodes effort in the model ID).

Where you would ask the user ordinary clarification questions, collect every question, uncertainty, assumption, edge case, and objection into one self-contained prompt instead. Make exactly one adversary call per invocation, carrying the original task, relevant evidence and constraints, and the complete question block. Ask it to answer every question, surface important ones you missed, attack weak assumptions, and give brutally honest recommendations. If the call fails, ask the user the question block directly.

Treat the response as serious advice; the decisions are yours. Report the adversary used and the decisions made, then continue the original task. Still ask the user when a genuine blocker remains or a decision is hard to undo or changes user-facing behavior or scope.
