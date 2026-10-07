---
name: implementer
description: Delegate an implementation, analysis, or review task to the configured worker model via the delegate script. Use when the user invokes /implementer or a /lead session delegates work.
disable-model-invocation: true
---

# Implementer

The script is the whole interface.

1. **Write the prompt** to `_scratch/implementer/<task>.md` (gitignored; the script keeps its reports and the worker's progress file there too). The worker has zero conversation context, so include: goal and constraints; relevant files and repo context; scope boundaries — what stays untouched; verification commands whose results must be reported; and the acceptance criteria you will judge by. The script appends the progress-file and fan-out instructions.

2. **Run**, in the background for anything over a couple of minutes. A quiet worker is normal — heartbeats and timeouts are the script's job.

```bash
uv run ~/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --worker <name> --prompt-file <file> [--worktree] [--subagents N]
```

Pick `--worker` by fit from `delegate.py --list`, which prints each worker's description. `--subagents N` fans any worker out to N sub-agents inside the run and returns one synthesized result; use it whenever the work has independent slices.

One handoff = one behavioral slice with one set of acceptance criteria and targeted verification. A prompt is too big at more than one numbered goal, more than ~5KB, or "and" joining independent surfaces; split it into sequential handoffs in the same authoritative checkout, inspecting each result before starting the next. The caller owns integration and final verification.

Parallel workers each need `--worktree`; worktrees start at HEAD, so commit anything workers must see (the JSON includes the path).

3. **Accept from the JSON.** Check `summary` (`summary_source`: progress file, stdout, or none) and `diff_stat`; open changed files selectively. `report_file` is the raw runner transcript, often megabytes: search it for the specific error, test name, or tail when the summary is missing, contradicts the diff, or the run failed. A timed-out run still reports its last progress state; judge what got done before revising or re-running.

`completed` is the worker's self-report: verify against the acceptance criteria and resume any remaining work. On `ok: false`, surface the exact error and stop; the user decides how to proceed.

Revise with a compact follow-up in the same cwd/worktree listing only the failed criteria, files/lines, error excerpts, and what stays unchanged.

Risky diffs get an independent delegated review — this prompt plus task-specific context:

```
Review these changes for bugs, regressions, missing tests, security issues, and requirements mismatches.

Lead with findings, each with severity, file:line, concrete failure mode, and fix direction. Report findings only — the caller applies fixes. If nothing is substantive, say so and name any residual test gaps.
```

Wrapper sub-agents (parallel fan-out triage or hosts without shell access): cheapest model, prompt = "run this exact delegate.py command and return its JSON output verbatim."
