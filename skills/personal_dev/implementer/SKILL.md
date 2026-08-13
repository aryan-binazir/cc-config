---
name: implementer
description: Delegate an implementation, analysis, or review task to the configured worker model via the delegate script. Use when Ar invokes /implementer or a /lead session delegates work.
---

# Implementer

The script is the whole interface — it owns config, models, runner flags, timeouts, and capture.

1. **Write a self-contained prompt** to `_scratch/prompts/<task>.md` (gitignored). The worker has zero conversation context. Include: goal and constraints; relevant files and repo context; scope boundaries — what stays untouched; verification commands whose results must be reported; the acceptance criteria you will judge by (stating them is the readiness test); and verbatim: end your output with a `## SUMMARY` section, max 15 lines, covering what changed, what was not done, verification results, and open questions.

2. **Run in the background** for anything expected to exceed a couple of minutes. A quiet worker is normal: heartbeats hit stderr every 60s, and the timeout defaults to 25 minutes.

```bash
uv run ~/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --tier <xhigh|high|medium|low> --prompt-file <file> [--worktree]
```

Tier: `low` simple mechanical, `medium` bulk with a clear spec, `high` hard or subtle, `xhigh` hardest. Parallel workers each need `--worktree`; worktrees start at HEAD, so commit anything workers must see (the JSON includes the path).

3. **Accept from the JSON.** Check the summary and diff stat; open changed files selectively, and the full report when the summary is missing or suspicious. Risky diffs: delegate an independent review (prompt in the lead skill). Revisions: a compact follow-up in the same cwd/worktree — failed criteria, files/lines, error excerpts, what stays unchanged — ending with the same `## SUMMARY` instruction.

On `ok: false`, surface the exact error and stop; Ar decides how to proceed.

Wrapper sub-agents are for parallel fan-out triage or hosts without shell access: cheapest model, prompt = "run this exact delegate.py command and return the report file contents."
