---
name: prove-it
description: >-
  Test what shipped for real: fresh main, throwaway full stack, one strong
  sub-agent per flow, every defect fixed through a reviewed PR,
  one consolidated report. Use when the user invokes
  /prove-it, says "prove it", "test everything we shipped", "shakedown",
  "pre-launch check", or "make sure it works for real".
---

# Prove It

`/prove-it <scope>` — test it for real, fix what breaks, hand back reviewed PRs.
You orchestrate: sub-agents test and fix, you read every diff yourself.

## Loop

1. **Fresh.** Pull latest `main` in every repo in scope and build from it.
2. **Stack.** `verify-sandbox`: ephemeral Postgres/Redis via `sbx`, the real
   services host-side, a stand-in auth provider that mints tokens for
   unlimited synthetic users, the real frontend dev server proxied to it.
   Zero human accounts needed.
3. **Brief.** One shared tester brief: env, helpers, the hard rules, the
   report shape.
4. **Fan out.** One sub-agent per flow or per shipped fix (Opus, highest
   reasoning). Browser testers drive the real UI with Playwright: stub the
   auth SDK in-page and inject bearer tokens so the real SPA renders. Anything
   touching money, scores, or irreversible state also gets an adversarial
   verifier. A log monitor watches for ERROR/panic/5xx the whole time.
5. **Triage.** By-design and cosmetic items get listed. Each real defect gets
   one `auto-implementer` run: own worktree, `tdd`, project checks,
   `verify-sandbox` results posted as a PR comment *before* `rocket-review`,
   reviewers in the foreground, findings patched, "What users will see"
   bullets in the PR body. A worker that times out mid-review resumes in the
   same worktree. Optional: `call-codex` on the fix plan first; it catches
   fixes that contradict specs.
6. **Merge.** Stop and ask the user; merge reviewed PRs only if they agree.
7. **Report.** One consolidated report, sandbox torn down clean, every
   checkout clean.

## Hard rules (verbatim in the brief)

- Own users, own tenants, own data: create yours, touch only yours.
- The checkout is read-only; harness code lives in `_scratch/`.
- Shared services keep running; stop only what you started.
- SQL reads state or moves a time gate that has no product path; everything
  else goes through real HTTP or the UI first.
- Every finding carries severity P0-P3, CONFIRMED or SUSPECTED, and file:line.

## Report shape

```
CLAIM <name>: VERIFIED | FAILED | NOT VERIFIED — evidence
PR #n: what users will see
OPEN: P<n> CONFIRMED|SUSPECTED file:line — one line each
SANDBOX: down clean · CHECKOUTS: clean
```
