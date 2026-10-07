---
name: verify-sandbox
description: >-
  Ephemeral verification stack (Postgres, Redis, any container): stand it up,
  run throwaway harness code against it, tear it down verified-clean. Use when
  verifying code end-to-end against a real database or service, when a verify
  step needs evidence beyond unit tests, or when asked to sandbox or spin up a
  throwaway stack or database.
disable-model-invocation: false
---

# Verify Sandbox

Prove the code works against real infrastructure, then vanish without a trace.

The tool is `<skill-dir>/scripts/sbx` (`sbx --help` lists every command). Configuration loads `verify-sandbox.local.yaml` when present, otherwise `verify-sandbox.example.yaml`, then applies `SBX_*` overrides; `sbx doctor` reports the result.

## Lane

- Prefer the configured Docker-compatible runtime; when infeasible, explain why and use an isolated alternative. Preserve unrelated host workloads.
- Every container carries `sbx=1` and `sbx.key=<key>`: create through `sbx` (add both labels when calling the runtime directly) and remove through `sbx down` / `sbx gc`, which reach labeled containers only.
- Data is synthetic: migrations, fixtures, generated rows. Real data enters a sandbox only on the user's explicit opt-in for that run.
- Harness code is throwaway and lives in `_scratch/` or the session scratchpad; a gap worth keeping becomes a proper test in the repo's suite.
- One key per task (ticket key or slug, e.g. `abc-42`) names the containers and is the unit of teardown.
- Temporary localhost services under test belong to the sandbox: start, drive, and end them within the run. The machine's long-lived dev servers keep running as found.

## Workflow

Apply setup and teardown steps to the infrastructure used.

1. **Up.** `sbx pg <key>` / `sbx redis <key>`; point the code under test at the printed URLs via env vars or flags, leaving committed config untouched.
2. **Migrate and seed.** The repo's real migrations, then synthetic data sized to the behavior under test.
3. **Exercise.** Drive each acceptance claim — happy path plus at least one failure or edge path — via the real binary, integration tests, or a throwaway harness. Always write and run a harness proving the changed behavior, driving any UI through a browser or PTY. Delegate building it through the `implementer` skill (`medium` fits most), then run it against the sandbox yourself: execution, evidence, and verdict are the main agent's own work.
4. **Evidence.** Capture the commands and their decisive output (query results, responses, exit codes) while the sandbox is still up.
5. **Down.** `sbx down <key>`; its clean confirmation is the teardown proof. For alternatives, stop temporary processes and verify cleanup. `sbx gc` reaps stale sandboxes.

## Report

Start with `RESULT: PASS` only when every acceptance claim is evidenced and cleanup is verified; otherwise `RESULT: FAIL`. In chat, evidence-first: each claim with its command and decisive output; open questions the sandbox left unanswered; teardown proof. When the task has a PR, also post a comment in collapsed `<details>` with `Sandbox: PASS` or `Sandbox: FAIL` in `<summary>`, including every test's method, result, and evidence, plus teardown confirmation.
