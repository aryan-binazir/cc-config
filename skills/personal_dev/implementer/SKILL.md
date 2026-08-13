---
name: implementer
description: Delegate an implementation, analysis, or review task to the configured worker model via the delegate script. Use when Ar invokes /implementer or a /lead session delegates work.
---

# Implementer

All mechanics (config, runner flags, timeouts, capture) live in the script and models live only in the lead YAML — never read the `call-*` skills, assemble runner commands, or hardcode model names.

1. **Write a self-contained prompt** to `_scratch/prompts/<task>.md` (gitignored; nothing unrecoverable). The worker has zero conversation context. Include: goal and constraints; relevant files and repo context; what must NOT be touched; verification commands whose results must be reported; the concrete acceptance criteria you will judge by (if you cannot state them, the task is not ready to delegate); and verbatim: end your output with a `## SUMMARY` section, max 15 lines, covering what changed, what was not done, verification results, and open questions.

2. **Run in the background** — never foreground-block on a run expected to exceed a couple of minutes. A quiet worker is normal: heartbeats hit stderr every 60s, and the YAML timeout defaults to 25 minutes.

```bash
uv run ~/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --tier <xhigh|high|medium|low> --prompt-file <file> [--worktree]
```

Tier: `low` simple mechanical, `medium` bulk work with a clear spec, `high` hard or subtle, `xhigh` hardest problems. Parallel workers each need `--worktree` (the JSON includes the path).

3. **Accept from the JSON, not the worker's word.** Check both the `## SUMMARY` and the diff stat; open changed files selectively, and the full report only when the summary is missing or suspicious. For risky diffs, optionally delegate an independent review (prompt lives in the lead skill). For revisions, send a compact follow-up prompt in the same cwd/worktree: failed criteria, files/lines, short error excerpts, what must stay unchanged; ask for a fresh `## SUMMARY`.

On `ok: false`, surface the exact error and stop — the invoker never implements the task itself; a Lead reports the failure to Ar and waits.

Wrapper sub-agents are only for parallel fan-out triage or hosts where the lead cannot run shell commands: cheapest model, prompt = "run this exact delegate.py command and return the report file contents."
