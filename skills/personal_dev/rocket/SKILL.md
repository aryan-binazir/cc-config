---
name: rocket
description: >-
  Ship a Linear/Jira issue or no-ticket task end to end: critique the plan,
  build test-first, review, babysit the PR, and merge on `lift-off`. Use when
  the user invokes rocket.
disable-model-invocation: true
---

# Rocket

Take a task from its current state to a reviewed, green PR.

Inputs, ahead of the task:

- **Profile:** `gpt` selects the GPT profile; otherwise `claude`.
- **`lift-off`:** authorizes marking the PR ready and merging once the merge
  gate passes, whether given up front or mid-run.
- **Task** (required): an issue key or URL from the configured tracker makes
  tracked work; anything else is no-ticket work, defined by the request and
  the conversation leading up to it.
- **Branch** (optional): honored exactly. Default: `amb/<issue-key>`,
  or `amb/<task-slug>` (short kebab-case) for no-ticket work.

Examples: `$rocket BBA-359` · `$rocket gpt lift-off BBA-359` ·
`$rocket fix stale cache invalidation`

Throughout: write the failing test before production code. After alignment,
own every call: decide, state it as an assumption, and keep moving. Stop and
ask only when you cannot continue: the change can't be proven, the merge gate
can't pass and you can't fix it, the next step touches data, production,
anything outside the repo, closing a PR, or another branch, or another PR makes
this one obsolete. Ask through the harness question tool (`AskUserQuestion`,
`request_user_input`) with full context: the PR link, the blocker, and options
with your recommendation first.

## Config

Resolve `<rocket-skill-dir>` (this `SKILL.md`'s absolute directory) and run:

```bash
uv run --script "<rocket-skill-dir>/scripts/resolve_config.py" <claude|gpt>
```

Its output is the only config source; stop on failure. Run the critic through
the `call-claude`, `call-codex`, or `call-cursor` skill matching its `runner`,
passing configured `model`, `effort`, `reasoning_effort`, and `timeout_ms` as
`--model`, `--effort`, `--reasoning-effort`, and `--timeout-ms`; omit absent
ones. Use exactly the configured runner and model; stop when either is
unavailable.

## 0. Pick Up The Current State

Own the work from its current state. Read the task and repo instructions, then
resume at the first step whose result is missing:

- nothing started → 1
- work on the default branch → `git switch -c <branch>`, then as below
- feature branch with work, no PR → finish the change test-first, then 5
- PR without a Rocket Review summary → 6
- reviewed PR → 7, verifying commits pushed after its summary and reporting
  them as unreviewed

Past step 1, the current branch and checkout are the resolved branch and
`checkout_path`. Report where Rocket entered and which steps ran before it.

## 1. Prepare The Checkout First

Checkout setup is the first state-changing action, before the full issue read,
briefing, planning, or code exploration.

1. Resolve the target repo: for tracked work, verify the issue key through the
   configured tracker's skill or connector, reading only enough to route; for
   no-ticket work, use the task context.
2. Run the helper with the resolved checkout mode, keyed by the issue key or,
   for no-ticket work, `NO-TICKET-<TASK-SLUG>`:

   ```bash
   uv run --script "<rocket-skill-dir>/scripts/ensure_branch.py" \
     --repo <absolute-repo-path> \
     --ticket-key <ISSUE-KEY-OR-NO-TICKET-KEY> \
     --branch-name <branch> \
     --checkout-mode <resolved-checkout> \
     --base-branch main
   ```

   `--branch-name`: a user-supplied branch exactly; always
   `amb/<task-slug>` for no-ticket work, keeping the synthetic key
   out of the branch; omitted for tracked work without a supplied branch, so
   the helper derives the default.
3. Require `ok: true`, `checkout_mode` matching config, and `branch` equal to
   the expected branch — the **resolved branch**. The returned absolute
   `checkout_path` is authoritative, even outside the default
   `<repo>/_scratch/worktrees/<ticket-key>`.
4. Tell the user the checkout mode, resolved branch, and path.
5. Stop and ask on a dirty target, path collision, unavailable `main`, failed
   setup, branch checked out elsewhere in `branch` mode, mode or branch
   mismatch, or a checkout off the resolved branch.

Every later step runs in `checkout_path`; hand delegates that exact path.

Read the full issue body (for no-ticket work, the task is the source of
truth), then the repo's instructions, relevant code, tests, docs, and git state
until the goal, accepted behavior, boundaries, and validation target are clear.

When repo rules require `_scratch/_context/<ticket-key>.md`, key it by the
issue or, for no-ticket work, the task slug.

## 2. Align

When the conversation already settled the task, state the problem, outcome,
and scope in a few lines and continue. Otherwise brief the user — **Problem**,
**Outcome**, **Scope and constraints** — and continue once they confirm or
correct it.

Settle open questions now, while the user is here. Ask only what inspection
can't answer and could change scope, acceptance criteria, user-facing behavior,
API or data contracts, the public test seam, or hard-to-reverse architecture:
one question at a time, three at most. State reversible assumptions and obvious
test seams. When material ambiguity survives three questions, say the task is
short of implementation-ready and ask every remaining material question at
once. From here, run autonomously.

## 3. Plan And Critique

Write a concise plan: behavior, affected areas, test seams, red-green slices,
and verification.

Send the resolved `critic` the complete task, the plan, the repo path, and repo
instructions. Keep it read-only and ask, in order:

1. Which parts are unnecessary, and what can be cut?
2. What simpler or repo-native approach fits?
3. Which code outside the plan depends on what changes?
4. What gaps and risks remain?

Run tests and baseline checks while it works; start production code once its
critique is folded in, stating each call it prompts as an assumption. One
round, unless the run fails or the user asks for more.

## 4. Implement Test-First

Read the `tdd` skill fully and follow it, with Rocket's seam rule replacing its
seam confirmation. Stay within the plan and scope; meet new ambiguity with a
stated assumption.

When the change has a visible frontend result and the environment can capture
it, take before/after screenshots for the PR.

## 5. Verify, Commit, And Push

Run targeted tests plus every validation the repo requires. Fix relevant
failures; report unrelated or pre-existing ones. When a real database or
service stack proves the change best, follow `verify-sandbox`.

Confirm you're on the resolved branch, commit per repo conventions, push to
`origin/<resolved-branch>` (setting upstream), and confirm the branch is
**synced**: upstream at local `HEAD`.

## 6. Review

Follow `rocket-review` with the resolved `review_profile.name`, the issue or
task as its spec, and any screenshots to attach. It opens the draft PR and owns
review. Any `[open: blocker]` it leaves holds the merge and leads the report.

## 7. Babysit

Follow `babysit-pr` on the PR.

- **Default:** keep the draft; babysit until checks are green, the bot window
  passed, and comments are handled.
- **`lift-off`:** mark it ready, babysit to the merge gate, and merge. An
  assumption that changed user-facing behavior or scope holds the merge for the
  user.

## Report

With a tracker ticket, follow `context-sync`. Then report, leading with every
assumption made after alignment: checkout path, branch, PR URL, delivered
behavior, commits, verification and screenshots, review ledger, babysit
outcome, and caveats, ending with the merge commit or the PR's open state.
