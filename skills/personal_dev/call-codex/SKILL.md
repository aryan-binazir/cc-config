---
name: call-codex
description: Use this whenever the user asks to call Codex, run Codex headlessly, invoke /call-codex, or get a second opinion from Codex. This skill gives the exact local command style for non-interactive Codex execution.
---

# Call Codex

Ask Codex for a second opinion, plan critique, implementation critique, or independent read on a prompt.

## Command

Use the bundled wrapper. It merges `call-codex.example.yaml` with the ignored
`call-codex.local.yaml`, then applies the resolved model, reasoning effort, and
timeout. It runs Codex with workspace sandboxing, automatic approval review,
and stdin redirected from `/dev/null`, defaulting to `gpt-5.6-sol` with `high`
reasoning effort:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
uv run --script "<call-codex-skill-dir>/scripts/call.py" "$PROMPT"
```

Inspect the effective config without calling Codex:

```bash
uv run --script "<call-codex-skill-dir>/scripts/call.py" --resolve --pretty
```

When the user explicitly requests a different model or reasoning effort, pass
`--model` or `--reasoning-effort` for that call and leave the config untouched.

`PROMPT` is the worker's entire context — make it self-contained: the question
or critique target, relevant files and repo context, and the output format you
want.

The resolved `timeout_ms` defaults to 15 minutes. Quiet periods are normal —
keep waiting.
