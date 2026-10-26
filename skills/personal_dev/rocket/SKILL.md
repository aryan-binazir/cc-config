---
name: rocket
description: >-
  Ship a Linear/Jira issue or no-ticket task end to end: clarify, critique the
  plan, build test-first, verify, push, and run configured review. Use when the
  user invokes rocket or rocket gpt.
disable-model-invocation: true
---

# Rocket

Take a reasonably specified task from intake to a reviewed PR, without a
persisted contract.

Inputs:

- **Profile**: `gpt` when the user passes a `gpt` argument before the task;
  otherwise `claude`.
- **Task** (required): an issue key or URL from the configured tracker makes
  tracked work; anything else is no-ticket work, defined by the request and
  the conversation leading up to it.
- **Branch** (optional): honored exactly. Default: `amb/<issue-key>`,
  or `amb/<task-slug>` (short kebab-case) for no-ticket work.

Throughout: resolve material ambiguity before acting, run every configured
critique, write the failing test before production code, and merge only on the
user's explicit request.

Examples: `$rocket BBA-359` · `$rocket gpt BBA-359` ·
`$rocket fix stale cache invalidation`

## Config

Before interpreting the task, resolve `<rocket-skill-dir>` (the absolute
directory of this `SKILL.md`) and run, passing the profile (`claude` or `gpt`)
as the positional argument:

```bash
uv run --script "<rocket-skill-dir>/scripts/resolve_config.py" [profile]
```

Stop on any resolver failure. Checkout mode, tracker, runners, models, and
effort come only from the resolved `plan_profile.config`.

Invoke `claude`, `codex`, and `cursor-agent` runners through the matching
`call-claude`, `call-codex`, or `call-cursor` skill, passing configured
`model`, `effort`, `reasoning_effort`, and `timeout_ms` as its `--model`,
`--effort`, `--reasoning-effort`, and `--timeout-ms`; omit absent ones. The
configured runner and model are the only acceptable choice: stop if either is
unavailable.

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
4. Immediately tell the user the checkout mode, resolved branch, and path.
5. Stop and ask the user on a dirty target, path collision, unavailable `main`,
   failed setup, branch checked out elsewhere in `branch` mode, mode or branch
   mismatch, or a checkout off the resolved branch.

Every later step runs in `checkout_path`; hand delegates that exact path and
keep them there.

Now read the full issue body (for no-ticket work, the task description is the
source of truth), then the repo's instructions, relevant code, tests, docs,
and git state until the goal, accepted behavior, boundaries, and validation
target are clear.

When repo rules require `_scratch/_context/<ticket-key>.md`, key it by the
issue or, for no-ticket work, the task slug.

## 2. Brief, Align, And Clarify

Brief the user from the task and repo evidence:

- **Problem:** what is wrong or missing.
- **Outcome:** what the task makes true.
- **Scope and constraints:** boundaries, acceptance criteria, and repo
  constraints shaping the implementation.

Then ask: is this the right direction, or should anything change first?
Continue once the user confirms or corrects it, re-inspecting evidence the
corrections touch. This gate sits outside the question limit below.

After alignment, run autonomously — the plan needs no approval — pausing only
for the decisions and blockers this workflow names.

Ask only what inspection can't answer and could materially change scope,
acceptance criteria, user-facing behavior, API or data contracts, the public
test seam, or hard-to-reverse architecture:

- One question at a time, three at most by default.
- State reversible assumptions and proceed.
- State obvious public test seams and proceed; confirm unclear ones.
- If material ambiguity survives three questions, say the task is short of
  implementation-ready and ask whether to keep clarifying or proceed on
  explicit assumptions.

Hard-to-undo, user-facing, or scope-changing decisions always go to the user.

## 3. Plan And Get Configured Critique

Write a concise plan: intended behavior, affected areas, test seams, red-green
slices, and required verification.

Have the resolved `critic` critique it against the task, repo evidence, and
repo instructions: give it the complete task, ask for concrete gaps, risks,
needless complexity, and simpler repo-native alternatives, and keep it
read-only.

Fold in actionable feedback; take material decisions to the user and state
reversible assumptions. One round, unless the run fails or the user asks for
more.

## 4. Implement Test-First

Read the `tdd` skill fully and follow it, with Rocket's seam rule replacing
its seam confirmation. Slice vertically through the chosen public seams: one
failing behavior test, watch it fail as expected, just enough production code
to pass, repeat.

Stay within the plan and scope, and stop on any new material ambiguity.

## 5. Verify, Commit, And Push

Run targeted tests plus every validation the repo requires. Fix relevant
failures; report unrelated or pre-existing ones. When a real database or
service stack proves the change best, follow `verify-sandbox`.

Just before committing, confirm the current branch is the resolved branch.
Commit per repo conventions, push explicitly to the resolved branch on `origin`,
setting upstream as needed, and confirm the branch is **synced**: upstream
`origin/<resolved-branch>` at local `HEAD`.

Rocket always commits and pushes and otherwise leaves PRs untouched; creation
belongs to Rocket Review.

## 6. Run Configured Review

**`rocket-review` runner:** follow the `rocket-review` skill with the resolved
`review_profile.name` and the issue or task description as its spec source. It
owns PR creation and resolution and replaces the verdict loop below.

**Other runners:** stop unless a PR exists. Have the resolved `review` runner
review the PR diff, given the full issue or task description, repo path, base
and head commits, PR URL, repo instructions, changed files, and verification
results. Require it to stay read-only, list only concrete actionable findings,
and end with exactly one verdict, defined in the prompt. Handle each literally:

- `APPROVED` or `NO ACTIONABLE FEEDBACK` (no fixes needed): finish.
- `APPROVED WITH FIXES` (a complete, enumerated fix list needing no
  re-review): apply every fix, rerun relevant verification, commit, push,
  confirm synced, finish.
- `CHANGES REQUESTED` (the reviewer must inspect the fixes): apply them, rerun
  relevant verification, commit, push, confirm synced, and have the same
  reviewer review the new PR diff; repeat until a terminal verdict.

Only an exact token is a verdict — friendly prose and a lack of severe findings
are not approval. Retry a missing or malformed verdict once with the required
format, then stop and report the blocker.

## Completion

Confirm the branch is synced, then report checkout mode and path, branch, PR
URL, delivered behavior, commits, verification, review result, and caveats.
Merge only on the user's explicit request.
