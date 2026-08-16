---
name: explorer
description: Delegate read-only codebase exploration — mapping a subsystem, answering "where/how does X happen", surveying call sites — to the fan-out explore worker via the delegate script. Use when the user invokes /explorer, or when a /lead or /rocket session wants recon without spending its own context.
---

# Explorer

The script is the whole interface.

1. **Write the question** to `_scratch/explorer/<topic>.md` (gitignored; the script keeps its reports and findings there too, and sweeps files older than a week): the question, why it matters (so the worker knows what counts as an answer), starting points if you have them, and the evidence you want (file:line, call paths, data flow). The worker has zero conversation context. The script appends the findings-format and fan-out instructions.

2. **Run**, in the background when the question is broad:

```bash
uv run ~/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --worker explore --prompt-file <file> [--subagents N]
```

The explore worker runs read-only: a mid-effort orchestrator that fans the reading out to sub-agents (default width in config; `--subagents N` sets it). Width follows breadth: `1` for a single pointed question, `3`+ for "map this subsystem" or several independent questions. Several explorers may run at once in the same checkout. A quiet worker is normal — heartbeats and timeouts are the script's job.

3. **Read the findings.** The JSON `summary` is the `## SUMMARY` tail; `summary_file` holds the full findings and is the deliverable — cite it in the next handoff instead of re-transcribing. `report_file` is the raw runner transcript, for diagnosis when a run fails or the findings look wrong. On `ok: false`, surface the exact error and stop.

Reach for your runtime's own quick explore subagent when the answer is needed in-context within seconds; reach for this skill when the exploration is broad, feeds a delegated handoff, or should run in the background while planning continues.
