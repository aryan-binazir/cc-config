---
name: call-claude
description: Use this whenever the user asks to call Claude, run Claude Code headlessly, invoke /call-claude, or get a second opinion from Claude. This skill gives the exact local command style for non-interactive Claude execution.
---

# Call Claude

Ask Claude Code for a second opinion, plan critique, implementation critique, or independent read on a prompt.

## Command

Use the bundled wrapper. It merges `call-claude.example.yaml` with the ignored
`call-claude.local.yaml`, then applies the resolved model, effort, and timeout.
It runs Claude in print mode with Auto permission review, defaulting to
`claude-opus-5` with `high` effort:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
uv run --script "<call-claude-skill-dir>/scripts/call.py" "$PROMPT"
```

Inspect the effective config without calling Claude:

```bash
uv run --script "<call-claude-skill-dir>/scripts/call.py" --resolve --pretty
```

When the user explicitly requests a different model or effort, pass `--model`
or `--effort` for that call and leave the config untouched.

`PROMPT` is the worker's entire context — make it self-contained: the question
or critique target, relevant files and repo context, and the output format you
want.

The resolved `timeout_ms` defaults to 15 minutes. Quiet periods are normal —
keep waiting.
