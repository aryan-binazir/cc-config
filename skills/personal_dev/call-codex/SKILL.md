---
name: call-codex
description: Call Codex headlessly for a second opinion, critique, or independent read. Use when the user asks to call, run, or consult Codex.
---

# Call Codex

The bundled wrapper runs `codex exec` in the workspace-write sandbox under Auto-review, with model, reasoning effort, and timeout from `call-codex.local.yaml` when present, otherwise `call-codex.example.yaml`:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
uv run --script "<call-codex-skill-dir>/scripts/call.py" "$PROMPT"
```

`--resolve --pretty` prints the effective config without calling Codex. When the user explicitly requests a model or reasoning effort, pass `--model` or `--reasoning-effort` for that call; leave the config as is.

`PROMPT` is the worker's entire context: the question or critique target, relevant files and repo context, and the output format you want.

Quiet periods are normal up to `timeout_ms` (30 minutes by default); keep waiting.
