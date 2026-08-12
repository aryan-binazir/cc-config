---
name: call-cursor
description: Use this whenever the user asks to call Cursor, call Composer, run cursor-agent headlessly, invoke /call-cursor, or get a second opinion from Cursor/Composer. This skill gives the exact local command style for non-interactive Cursor execution.
---

# Call Cursor

Use this skill when the task is to ask Cursor/Composer for a second opinion, plan critique, implementation critique, or independent read on a prompt.

## Command

Use the bundled wrapper. It merges `call-cursor.example.yaml` with the ignored
`call-cursor.local.yaml`, then applies the resolved model and timeout to the
call:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
uv run --script "<call-cursor-skill-dir>/scripts/call.py" "$PROMPT"
```

The wrapper invokes Cursor in print mode with sandboxing and no force bypass. It
defaults to `cursor-grok-4.6-xhigh` (Grok 4.6 Extra High); Cursor carries the
effort/speed variant in the model ID rather than a separate effort option, so
the `xhigh` suffix *is* the reasoning effort. Always call Grok 4.6 at extra-high
effort — never `cursor-grok-4.6-high`, `-medium`, `-low`, or any 4.5 variant.

Inspect the effective config without calling Cursor:

```bash
uv run --script "<call-cursor-skill-dir>/scripts/call.py" --resolve --pretty
```

Cursor CLI Auto-review must be configured and supported by the installed CLI.
Stop if it is unavailable; never fall back to a bypass mode.

## Model Selection

Default to the configured model, `cursor-grok-4.6-xhigh`, unless the user
explicitly specifies a different model. Pass a one-off choice with `--model`; do
not edit the config for it.

If the user asks for Grok without naming a version or effort, that means
`cursor-grok-4.6-xhigh`. To pass it explicitly:

```bash
cursor-agent --print --trust --sandbox enabled \
  --model cursor-grok-4.6-xhigh "$PROMPT"
```

Use `cursor-grok-4.6-xhigh-fast` only when the user explicitly asks for the fast
variant.

If the user specifies an exact model name, pass that exact model with `--model`.
If they name a model family without an exact tag, resolve it against
`cursor-agent --list-models` rather than guessing a tag.

## Prompt Guidance

Put the full task in `PROMPT`. Include:
- the question or critique target
- any relevant files, paths, or repo context
- the output format you want

Do not rely on Cursor to infer the task from surrounding conversation. The CLI process should receive enough context to complete the job on its own.

## Waiting

The resolved `timeout_ms` defaults to 15 minutes. Quiet periods are normal. Do
not stop early just because there has been no output for a few minutes.
