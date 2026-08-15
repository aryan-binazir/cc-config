---
name: verify-sandbox
description: >-
  Stand up an ephemeral verification stack (Postgres, Redis, arbitrary
  containers) in rootless podman, run throwaway harness code against it, and
  tear it down verified-clean. Use when verifying code end-to-end against a
  real local database or service, when a verify step needs evidence beyond
  unit tests, or when the user asks to sandbox or spin up a throwaway stack
  or database.
---

# Verify Sandbox

Prove the code works against real infrastructure, then vanish without a trace.

`<skill-dir>` is the directory containing this file; the tool is
`<skill-dir>/scripts/sbx` (`sbx --help` lists every command).

## Lane

- Rootless podman is the sandbox's whole world; everything else on the machine
  — docker, clusters, long-lived dev containers — stays exactly as found.
- Every container carries `sbx=1` and `sbx.key=<key>`: create through `sbx`
  (add both labels when calling podman directly), remove through
  `sbx down` / `sbx gc`, which reach labeled containers only.
- Data is synthetic: migrations, fixtures, generated rows. Real data enters a
  sandbox only on the user's explicit opt-in for that run.
- Harness code is throwaway and lives in `_scratch/` or the session
  scratchpad; a gap worth keeping becomes a proper test in the repo's suite.
- One lowercase key per task (ticket key or slug, e.g. `abc-42`) names the
  containers and is the unit of teardown.
- Temporary localhost services under test belong to the sandbox: start them,
  drive them, and end them within the run. The machine's long-lived dev
  servers keep running as found.

## Workflow

1. **Up.** `sbx pg <key>` / `sbx redis <key>` — random localhost ports;
   capture the printed URLs and point the code under test at them via env
   vars or flags, keeping committed config untouched.
2. **Migrate and seed.** The repo's real migrations, then synthetic data
   sized to the behavior under test.
3. **Exercise.** Drive each of the task's acceptance claims — happy path plus
   at least one failure or edge path — via the real binary, integration tests,
   or a throwaway harness. A real harness beats a thin probe: delegate
   harness construction through the `implementer` skill (`medium` fits most
   harness builds; pick the tier by fit). The main agent then runs that
   harness against the sandbox itself — execution, evidence, and the verdict
   are the main agent's own work.
4. **Evidence.** Capture the commands and their decisive output (query
   results, responses, exit codes) while the sandbox is still up.
5. **Down.** `sbx down <key>`; its clean confirmation is the teardown proof.
   `sbx gc` reaps sandboxes older than 4 hours.

## Report

Start with `RESULT: PASS` only when every acceptance claim is evidenced and
`sbx down` confirms clean teardown; otherwise start with `RESULT: FAIL`.
In chat, evidence-first: each claim with its command and decisive output;
open questions the sandbox left unanswered; the `sbx down` confirmation.
Complete means evidenced and torn down.
