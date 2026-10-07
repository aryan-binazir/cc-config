---
name: explorer
description: Delegate read-only codebase recon — mapping a subsystem, answering "where/how does X happen", surveying call sites — to the fan-out explore worker. Use when the user invokes /explorer, or a /lead or /rocket session wants recon without spending its own context.
disable-model-invocation: true
---

# Explorer

The script is the whole interface. Your runtime's own quick explore subagent suits answers needed in-context within seconds; this skill suits exploration that is broad, feeds a delegated handoff, or should run in the background while planning continues.

1. **Write the question** to `_scratch/explorer/<topic>.md` (gitignored, swept after a week; the script keeps reports and findings there too). The worker has zero conversation context, so include: the question, why it matters (so the worker knows what counts as an answer), starting points if you have them, and the evidence you want (file:line, call paths, data flow). The script appends the findings-format and fan-out instructions.

2. **Run**, in the background when the question is broad:

```bash
uv run ~/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --worker explore --prompt-file <file> [--subagents N]
```

The worker runs read-only and fans the reading out to sub-agents; `--subagents N` overrides the config width — `1` for one pointed question, `3`+ to map a subsystem or answer several independent questions. Several explorers can share one checkout. A quiet worker is normal — heartbeats and timeouts are the script's job.

3. **Read the findings.** The JSON `summary` is the `## SUMMARY` tail; `summary_file` holds the full findings and is the deliverable — cite it in the next handoff instead of re-transcribing. `report_file` is the raw runner transcript, for diagnosis when a run fails or the findings look wrong. `completed` is the worker's self-report: verify the findings against the question and evidence requested, and resume any remaining exploration. On `ok: false`, surface the exact error and stop.
