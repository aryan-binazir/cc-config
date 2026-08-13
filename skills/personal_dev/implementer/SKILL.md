---
name: implementer
description: Delegate an implementation, analysis, or review task to the configured worker model via the delegate script. Use when Ar invokes /implementer or a /lead session delegates work.
---

# Implementer

The script is the whole interface.

1. **Write a self-contained prompt** to `_scratch/prompts/<task>.md` (gitignored). The worker has zero conversation context. Include: goal and constraints; relevant files and repo context; scope boundaries — what stays untouched; verification commands whose results must be reported; the acceptance criteria you will judge by; and verbatim: end your output with a `## SUMMARY` section, max 15 lines, covering what changed, what was not done, verification results, and open questions.

2. **Run in the background** for anything expected to exceed a couple of minutes. A quiet worker is normal — heartbeats and timeouts are the script's job.

```bash
uv run ~/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --worker <name> --prompt-file <file> [--worktree]
```

`--list` shows the workers and what each is good at; pick by fit. Parallel workers each need `--worktree`; worktrees start at HEAD, so commit anything workers must see (the JSON includes the path).

3. **Accept from the JSON.** Check the summary and diff stat; open changed files selectively, and the full report when the summary is missing or suspicious. Revisions: a compact follow-up in the same cwd/worktree — failed criteria, files/lines, error excerpts, what stays unchanged — ending with the same `## SUMMARY` instruction.

On `ok: false`, surface the exact error and stop; Ar decides how to proceed.

Risky diffs get an independent delegated review — this prompt plus task-specific context:

```
Review these changes for bugs, regressions, missing tests, security issues, and requirements mismatches.

Prioritize findings over summary. For each finding include:
- severity
- file and line reference
- concrete failure mode
- suggested fix direction

Report findings only — the caller applies fixes. If there are no substantive findings, say so and name any residual test gaps.
```

Wrapper sub-agents (parallel fan-out triage or hosts without shell access): cheapest model, prompt = "run this exact delegate.py command and return the report file contents."
