---
name: call-cursor
description: Use this whenever the user asks to call Cursor, call Composer, run cursor-agent headlessly, invoke /call-cursor, or get a second opinion from Cursor/Composer. This skill gives the exact local command style for non-interactive Cursor execution.
---

# Call Cursor

Ask Cursor/Composer for a second opinion, plan critique, implementation critique, or independent read on a prompt.

## Command

Use the bundled launcher. It merges `call-cursor.example.yaml` with the ignored
`call-cursor.local.yaml`, then applies the resolved model and timeout. It runs
Cursor in print mode with sandboxing enabled, defaulting to
`cursor-grok-4.6-xhigh` (Grok 4.6 Extra High):

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
bash "<call-cursor-skill-dir>/scripts/call.sh" "$PROMPT"
```

Inspect the effective config without calling Cursor:

```bash
bash "<call-cursor-skill-dir>/scripts/call.sh" --resolve --pretty
```

The launcher passes `--auto-review` explicitly. Run only in the sandboxed,
approval-reviewed mode shown here; an older CLI that lacks Auto-review must fail
rather than falling back to another approval mode. In T3 Code, request host
execution for this exact launcher; a reusable approval may cover only
`bash <call-cursor-skill-dir>/scripts/call.sh`. The launcher then runs the full
wrapper through the user service manager so UV and Cursor escape the app's
filesystem and network restrictions while Cursor's own sandbox remains enabled.
Outside T3 Code, it runs the wrapper directly.

## Model Selection

Cursor carries effort in the model ID — the `xhigh` suffix *is* the reasoning
effort. Call Grok only as 4.6 at extra-high: `cursor-grok-4.6-xhigh`. That is
also what a bare "use Grok" request means. To pass it explicitly:

```bash
cursor-agent --print --trust --auto-review --sandbox enabled \
  --model cursor-grok-4.6-xhigh "$PROMPT"
```

Use `cursor-grok-4.6-xhigh-fast` only when the user explicitly asks for the
fast variant.

When the user requests a different model, pass their exact model name with
`--model` for that call and leave the config untouched. When they name a model
family without an exact tag, resolve it against `cursor-agent --list-models`.

`PROMPT` is the worker's entire context — make it self-contained: the question
or critique target, relevant files and repo context, and the output format you
want.

The resolved `timeout_ms` defaults to 15 minutes. Quiet periods are normal —
keep waiting.
