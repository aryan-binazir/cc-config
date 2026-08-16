---
name: explorer
description: Delegate read-only codebase exploration — mapping a subsystem, answering "where/how does X happen", surveying call sites — to the cheap fan-out explore worker via the delegate script. Use when Ar invokes /explorer, or when a /lead or /rocket session wants recon without spending its own context.
---

# Explorer

Same script and mechanics as the `implementer` skill (background runs, JSON result, `report_file`); read that once. Explorer differs in what goes into the prompt and what comes back.

1. **Write the question** to `_scratch/implementer/<topic>.md`: the question, why it matters (so the worker knows what counts as an answer), starting points if you have them, and the evidence you want (file:line, call paths, data flow). The worker has zero conversation context.

2. **Run**:

```bash
uv run ~/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --worker explore --prompt-file <file> [--subagents N]
```

The explore worker is read-only: a mid-effort orchestrator that fans the reading out to cheap sub-agents (default width in config; `--subagents N` sets it). Width follows breadth: `1` for a single pointed question, `3`+ for "map this subsystem" or several independent questions. Several explorers may run at once in the same checkout — no worktree needed.

3. **Read the findings.** The JSON `summary` is the `## SUMMARY` tail; `summary_file` holds the full findings and is the deliverable — cite it in the next implementer prompt instead of re-transcribing. `diff_stat` shows changes since the run's base commit, so entries beyond `preexisting_changes` mean the worker overstepped — discard that run.

Reach for the Claude `Explore` subagent instead when the answer is needed in-context within seconds; reach for this skill when the exploration is broad, feeds a delegated handoff, or should run in the background while planning continues.
