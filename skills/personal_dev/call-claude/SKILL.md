---
name: call-claude
description: Use this whenever the user asks to call Claude, run Claude Code headlessly, invoke /call-claude, or get a second opinion from Claude. This skill gives the exact local command style for non-interactive Claude execution.
---

# Call Claude

Use this skill when the task is to ask Claude Code for a second opinion, plan critique, implementation critique, or independent read on a prompt.

## Command

Use the bundled wrapper. It merges `call-claude.example.yaml` with the ignored
`call-claude.local.yaml`, then applies the resolved model, effort, and timeout
to the call:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
uv run --script "<call-claude-skill-dir>/scripts/call.py" "$PROMPT"
```

The wrapper invokes Claude in print mode with Auto permission review. It
defaults to `claude-opus-5` with `high` effort.

Inspect the effective config without calling Claude:

```bash
uv run --script "<call-claude-skill-dir>/scripts/call.py" --resolve --pretty
```

If the user explicitly requests a different model or effort, pass `--model` or
`--effort` to the wrapper for that call. Do not edit the config for a one-off
override.

## Prompt Guidance

Put the full task in `PROMPT`. Include:
- the question or critique target
- any relevant files, paths, or repo context
- the output format you want

Do not rely on Claude to infer the task from surrounding conversation. The CLI process should receive enough context to complete the job on its own.

## Waiting

The resolved `timeout_ms` defaults to 15 minutes. Quiet periods are normal. Do
not stop early just because there has been no output for a few minutes.
