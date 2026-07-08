---
name: implementer
description: Delegate an implementation, analysis, or review task to the configured worker model via the delegate script. Use when Ar invokes /implementer directly, or when a /lead session delegates work. Model choices live in the lead YAML config, never in this file.
---

# Implementer

Delegate a task to the configured worker. All mechanics (config resolution, runner flags, timeouts, report capture) live in the script — do not read the `call-*` skills or assemble runner commands yourself.

1. **Write a self-contained prompt to a file** at `_scratch/prompts/<short-task-name>.md` in the repo being worked on (create the directory if needed; `_scratch/` is gitignored by convention and Ar prunes it freely — never put anything unrecoverable there). The worker has zero conversation context. Include: the goal and constraints; relevant files, paths, and repo context; what must NOT be touched; verification commands to run (typecheck, lint, tests) with the requirement to report their results; and this exact reporting instruction: end your output with a `## SUMMARY` section, max 15 lines, covering what changed, what was not done, verification results, and open questions.

   Readiness test: write the concrete acceptance criteria you will judge the result by into the prompt. If you cannot state them, the task is not ready to delegate — go back to planning.

2. **Run the script in the background** and keep working — never foreground-block on a run expected to exceed a couple of minutes. The script heartbeats to stderr every 60s; a quiet worker is normal. Timeouts come from the YAML (default 25 minutes):

```bash
uv run /home/ar/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --tier <xhigh|high|medium|low> --prompt-file <file> [--worktree]
```

Tier: `low` for simple, low-risk mechanical work, `medium` for bulk/mechanical work with a clear spec, `high` for hard or subtle work, `xhigh` for the hardest problems. Parallel workers must each get `--worktree` so edits don't collide; the JSON output includes the worktree path.

3. **Accept from the JSON, not the worker's word.** The JSON includes the worker's `## SUMMARY` and a `git diff --stat` of the working directory. Always check both; open specific changed files selectively when something warrants it. Only open the full report file when the summary is missing or suspicious. For substantial or risky diffs, optionally delegate an independent review pass (the lead skill has the review prompt) before accepting. Never treat the worker's self-report as done.

   If the result is close but needs revision, run a new compact follow-up prompt in the same cwd/worktree (use the JSON `worktree` path when present). Keep feedback specific rather than exhaustive: failed acceptance criteria, relevant files/lines or behavior, short test/error excerpts, and anything that must stay unchanged. Ask for an updated `## SUMMARY`; do not paste full reports or diffs unless the worker needs that context.

If the script returns `ok: false`, surface the exact error and stop. There is no fallback: the invoker never implements the task itself — a Lead reports the failure to Ar and waits.

Wrapper sub-agents are not part of the normal path. Use one only for parallel fan-out where cheap first-pass triage of many reports is wanted, or in environments where the lead cannot run shell commands. Pick the cheapest sub-agent model the host offers; the wrapper's entire prompt is "run this exact delegate.py command and return the report file contents."
