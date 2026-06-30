---
name: call-cursor
description: Use this whenever the user asks to call Cursor, call Composer, run cursor-agent headlessly, invoke /call-cursor, or get a second opinion from Cursor/Composer. This skill gives the exact local command style for non-interactive Cursor execution.
---

# Call Cursor

Use this skill when the task is to ask Cursor/Composer for a second opinion, plan critique, implementation critique, or independent read on a prompt.

## Command

Use `cursor-agent` in print mode with the user's standard flags:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
cursor-agent -p -f --model composer-2.5 "$PROMPT"
```

The flags `-p -f` are the expected local convention for headless Cursor/Composer calls in these workflows. Use `--model composer-2.5` by default when no model is specified.

## Model Selection

Use `composer-2.5` unless the user explicitly specifies a different model.

If the user specifies Opus, use the current pinned Claude Opus model:

```bash
cursor-agent -p -f --model claude-opus-4-8-thinking-high "$PROMPT"
```

If the user specifies Sonnet, use the current CLI-exposed Sonnet 5 Extra High
alias:

```bash
cursor-agent -p -f --model claude-sonnet-5-xhigh "$PROMPT"
```

`cursor-agent --list-models` currently labels that alias as `Sonnet 5 1M Extra
High`. Do not use it when the user explicitly asks for a non-1M Sonnet model;
no accepted 300K Sonnet 5 Extra High tag has been found in the local CLI.

If the user specifies an exact model name, pass that exact model with `--model`.

## Prompt Guidance

Put the full task in `PROMPT`. Include:
- the question or critique target
- any relevant files, paths, or repo context
- the output format you want

Do not rely on Cursor to infer the task from surrounding conversation. The CLI process should receive enough context to complete the job on its own.

## Waiting

Allow up to 15 minutes for substantial work:

```text
900000 ms
```

Quiet periods are normal. Do not stop early just because there has been no output for a few minutes.
