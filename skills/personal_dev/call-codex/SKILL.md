---
name: call-codex
description: Use this whenever the user asks to call Codex, run Codex headlessly, invoke /call-codex, or get a second opinion from Codex. This skill gives the exact local command style for non-interactive Codex execution.
---

# Call Codex

Use this skill when the task is to ask Codex for a second opinion, plan critique, implementation critique, or independent read on a prompt.

## Command

Use the bundled wrapper. It merges `call-codex.example.yaml` with the ignored
`call-codex.local.yaml`, then applies the resolved model, reasoning effort, and
timeout to the call:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
uv run --script "<call-codex-skill-dir>/scripts/call.py" "$PROMPT"
```

The wrapper invokes Codex with workspace sandboxing, automatic approval review,
and stdin redirected from `/dev/null`. It defaults to `gpt-5.6-sol` with
`high` reasoning effort.

Inspect the effective config without calling Codex:

```bash
uv run --script "<call-codex-skill-dir>/scripts/call.py" --resolve --pretty
```

If the user explicitly requests a different model or reasoning effort, pass
`--model` or `--reasoning-effort` to the wrapper for that call. Do not edit the
config for a one-off override.

## Prompt Guidance

Put the full task in `PROMPT`. Include:
- the question or critique target
- any relevant files, paths, or repo context
- the output format you want

Do not make Codex infer the task from surrounding conversation. The CLI process should receive enough context to complete the job on its own.

## Waiting

The resolved `timeout_ms` defaults to 15 minutes. Quiet periods are normal. Do
not stop early just because there has been no output for a few minutes.
