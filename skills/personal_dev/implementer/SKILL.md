---
name: implementer
description: Delegate an implementation, analysis, or review task to the configured worker model via the delegate script. Use when Ar invokes /implementer or a /lead session delegates work.
---

# Implementer

The script is the whole interface.

1. **Write a self-contained prompt** to `_scratch/implementer/<task>.md` (gitignored; the script keeps its reports and the worker's summary file in the same directory). The worker has zero conversation context. Include: goal and constraints; relevant files and repo context; scope boundaries — what stays untouched; verification commands whose results must be reported; and the acceptance criteria you will judge by. Do not add summary instructions — the script appends them and tells the worker where to write its progress file.

2. **Run in the background** for anything expected to exceed a couple of minutes. A quiet worker is normal — heartbeats and timeouts are the script's job.

```bash
uv run ~/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --worker <name> --prompt-file <file> [--worktree]
```

Pick `--worker` by fit:

- `xhigh`: Escalation only — use after `high` fails, or when Ar explicitly requests maximum reasoning.
- `high`: Default for difficult work, including subtle bugs, architecture, gnarly debugging, and complex implementation.
- `medium`: Bulk implementation with a clear spec.
- `low`: Targeted, near-deterministic edits from an exact spec.
- `frontend`: Frontend and UI work.

One handoff = one behavioral slice with one set of acceptance criteria and targeted verification. Signals a prompt is too big: more than one numbered goal, more than ~5KB, or "and" joining independent surfaces (e.g. archive + import + idempotency + CI gate). Split those into sequential handoffs in the same authoritative checkout, inspecting each result before starting the next. Same rule for follow-ups: a revision lists the failed criteria, not a fresh multi-item review agenda. The caller owns integration and final verification.

Parallel workers each need `--worktree`; worktrees start at HEAD, so commit anything workers must see (the JSON includes the path).

3. **Accept from the JSON.** Check `summary` (`summary_source` says whether it came from the worker's progress file, its stdout, or nowhere) and `diff_stat`; open changed files selectively. `report_file` is the raw runner transcript — every command, its output, echoed diffs — often megabytes; it is for diagnosis when the summary is missing, contradicts the diff, or the run failed, and rewards searching for the specific error, test name, or tail you need. A timed-out run still reports the last progress-file state, so judge what got done before deciding to revise or re-run. Revisions: a compact follow-up in the same cwd/worktree — failed criteria, files/lines, error excerpts, what stays unchanged — the script appends the summary instruction again.

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

Wrapper sub-agents (parallel fan-out triage or hosts without shell access): cheapest model, prompt = "run this exact delegate.py command and return its JSON output verbatim."
