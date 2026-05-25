---
name: call-claude
description: Use this whenever the user asks to call Claude, run Claude Code headlessly, invoke /call-claude, or get a second opinion from Claude. This skill gives the exact local command style for non-interactive Claude execution.
---

# Call Claude

Use this skill when the task is to ask Claude Code for a second opinion, plan critique, implementation critique, or independent read on a prompt.

## Command

Use Claude in print mode with the user's standard skip-permissions flag:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
claude --dangerously-skip-permissions -p "$PROMPT"
```

This is the expected local convention for non-interactive Claude Code calls in these workflows.

## Prompt Guidance

Put the full task in `PROMPT`. Include:
- the question or critique target
- any relevant files, paths, or repo context
- the output format you want

Do not rely on Claude to infer the task from surrounding conversation. The CLI process should receive enough context to complete the job on its own.

## Waiting

Allow up to 15 minutes for substantial work:

```text
900000 ms
```

Quiet periods are normal. Do not stop early just because there has been no output for a few minutes.
