---
name: prove-it
description: >-
  Test what shipped for real on a throwaway full stack, fixing every defect
  through a reviewed PR. Use for a shakedown or pre-launch check, or when asked
  to prove it works for real.
---

# Prove It

`/prove-it <scope>` — test it for real, fix what breaks, hand back reviewed PRs.
Workers own setup, testing, and fixes end to end. You keep them moving,
resolve disputed findings, and review diffs.

## Loop

1. **Fresh.** Pull latest `main` in every repo in scope and build from it.
2. **Stack.** `verify-sandbox`: ephemeral Postgres/Redis via `sbx`, the real
   services host-side, a stand-in auth provider that mints tokens for
   unlimited synthetic users, the real frontend dev server proxied to it.
   Zero human accounts needed.
3. **Brief.** Share one environment and tester brief. Workers handle routine
   fixture setup within their ownership; reuse completed evidence.
4. **Fan out.** Run independent workers in parallel, one per complete user
   journey, grouping related checks, on the user's chosen model and reasoning
   level. Browser testers drive the real UI with Playwright: stub the auth SDK
   in-page and inject bearer tokens so the real SPA renders. Anything touching
   money, scores, or irreversible state also gets an adversarial verifier. A
   log monitor watches for ERROR/panic/5xx throughout.
5. **Triage.** List by-design and cosmetic items. Each real defect gets one
   `auto-implementer` run: own worktree, `tdd`, project checks,
   `verify-sandbox` results posted as a PR comment *before* `rocket-review`,
   reviewers in the foreground, findings patched, "What users will see"
   bullets in the PR body. Resume interrupted workers from their existing
   worktree and evidence. Optional: `call-codex` on the fix plan first; it
   catches fixes that contradict specs.
6. **Merge.** Ask the user; merge reviewed PRs on their yes.
7. **Report.** One consolidated report, owned resources cleaned up,
   unrelated work preserved.

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
OWNED RESOURCES: cleaned up · UNRELATED WORK: preserved
```
