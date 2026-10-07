---
name: call-claude
description: Call Claude Code headlessly for a second opinion, critique, or independent read. Use when the user asks to call, run, or consult Claude.
---

# Call Claude

The bundled wrapper runs Claude in print mode under Auto permission review, with model, effort, and timeout from `call-claude.local.yaml` when present, otherwise `call-claude.example.yaml`:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
uv run --script "<call-claude-skill-dir>/scripts/call.py" "$PROMPT"
```

`--resolve --pretty` prints the effective config without calling Claude. When the user explicitly requests a model or effort, pass `--model` or `--effort` for that call; leave the config as is.

`PROMPT` is the worker's entire context: the question or critique target, relevant files and repo context, and the output format you want.

Quiet periods are normal up to `timeout_ms` (30 minutes by default); keep waiting.
