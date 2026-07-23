---
name: call-codex
description: Use this whenever the user asks to call Codex, run Codex headlessly, invoke /call-codex, or get a second opinion from Codex. This skill gives the exact local command style for non-interactive Codex execution.
---

# Call Codex

Use this skill when the task is to ask Codex for a second opinion, plan critique, implementation critique, or independent read on a prompt.

## Command

Use `codex exec` with workspace sandboxing and automatic approval review:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
codex exec --sandbox workspace-write --ask-for-approval on-request \
  -c approvals_reviewer=auto_review "$PROMPT" < /dev/null
```

Keep stdin redirected from `/dev/null`. Codex can otherwise wait on or infer behavior from standard input in ways that make detached/headless calls less reliable.

## Prompt Guidance

Put the full task in `PROMPT`. Include:
- the question or critique target
- any relevant files, paths, or repo context
- the output format you want

Do not make Codex infer the task from surrounding conversation. The CLI process should receive enough context to complete the job on its own.

## Waiting

Allow up to 15 minutes for substantial work:

```text
900000 ms
```

Quiet periods are normal. Do not stop early just because there has been no output for a few minutes.
