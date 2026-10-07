---
name: call-cursor
description: Call Cursor/Composer (cursor-agent) headlessly for a second opinion, critique, or independent read. Use when the user asks to call, run, or consult Cursor or Composer.
---

# Call Cursor

The bundled launcher runs `cursor-agent --print` sandboxed under `--auto-review`, with model and timeout from `call-cursor.local.yaml` when present, otherwise `call-cursor.example.yaml`:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
bash "<call-cursor-skill-dir>/scripts/call.sh" "$PROMPT"
```

`--resolve --pretty` prints the effective config without calling Cursor.

Run only through this launcher; sandbox and Auto-review are mandatory, so when the CLI lacks Auto-review, fail and report rather than switch approval modes. In T3 Code, request host execution for this exact launcher (a reusable approval covers only `bash <call-cursor-skill-dir>/scripts/call.sh`); it escapes the app's filesystem and network limits via the user service manager while Cursor's own sandbox stays on.

## Model Selection

The model ID carries the effort: the `xhigh` suffix *is* the reasoning effort. Grok runs only as `grok-4.7-xhigh`, a bare "use Grok" included; `grok-4.7-xhigh-fast` only on explicit request. For any other requested model, pass its exact name with `--model` for that call, resolving a bare family name against `cursor-agent --list-models`; leave the config as is.

`PROMPT` is the worker's entire context: the question or critique target, relevant files and repo context, and the output format you want.

Quiet periods are normal up to `timeout_ms` (30 minutes by default); keep waiting.
