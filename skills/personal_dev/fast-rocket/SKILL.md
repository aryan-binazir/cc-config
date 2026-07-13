---
name: fast-rocket
description: >-
  Take a configured Linear or Jira issue or explicit no-ticket task plus an optional exact
  user-supplied branch through
  focused clarification, configured plan critique, test-driven implementation,
  verification, commit and push, draft PR creation, and configured review. Use this
  whenever the user invokes fast-rocket or asks for the lighter, lower-friction
  alternative to rocket-plan for an end-to-end task.
---

# Fast Rocket

Use this for a reasonably specified task that should move quickly from intake
to a reviewed PR. The local config selects Linear or Jira; when the user
explicitly says there is no ticket, accept a clear task description instead. It
is separate from Rocket: do not persist a Rocket contract, resolve Rocket plan
profiles, or wait for explicit approval of the implementation plan.

Expect one required task input and up to three optional inputs:

1. An issue ID or URL for the configured tracker, or an explicit `no ticket`
   task description.
2. Optionally, the exact branch name to use.
3. Optionally, the literal `implementer` modifier.
4. Optionally, the literal `grill` modifier.

For example:

`$fast-rocket BBA-359`

`$fast-rocket BBA-359 implementer`

`$fast-rocket BBA-359 grill`

`$fast-rocket no ticket: fix stale cache invalidation`

If the branch is omitted, derive `aryan-binazir/<resolved-issue-key>` for tracked
work or `aryan-binazir/<task-slug>` for explicit no-ticket work, using a
reasonable short kebab-case slug. If the user supplies a branch, honor it
exactly. Unless the user explicitly says there is no ticket, ask for an issue ID
or URL from the configured tracker. Do not infer another tracker. Treat
`implementer` and `grill` as modifiers, not as branch names.

Never guess past material ambiguity, skip the required external critiques,
write production code before a driving test, or merge unless the user asks.

## Config

Before interpreting the task input, resolve `<fast-rocket-skill-dir>` as the
absolute directory containing this `SKILL.md`, then run:

```bash
uv run --script "<fast-rocket-skill-dir>/scripts/resolve_config.py"
```

The resolver reads the required ignored `fast-rocket.local.yaml` beside this
skill. Stop on any resolver failure; do not choose a checkout mode, tracker,
runner, or model from prose or machine availability. The flat config provides
`checkout`, `tracker`, `critic`, optional `implementer`, optional `grill`, and
`review`. `checkout` is either `worktree` or `branch`. Model and effort values
come only from that resolved config.

For configured `cursor`, `claude`, or `codex` runners, read the matching
`call-cursor`, `call-claude`, or `call-codex` skill before invocation. Pass the
configured `model`, `effort`, `reasoning_effort`, and `timeout_ms` when present;
omit absent options so the runner uses its own defaults. Stop if a configured
runner or model is unavailable instead of silently substituting another.

Pass runner options using their native flags: Cursor `--model`, Claude
`--model` and `--effort`, and Codex `--model` plus
`-c model_reasoning_effort="<reasoning_effort>"`. Treat `timeout_ms` as the
maximum wait for the configured invocation, not as a runner CLI flag.

## 1. Prepare The Configured Checkout First

Resolve the shared branch helper relative to this skill as
`<fast-rocket-skill-dir>/../rocket/scripts/ensure_branch.py`.

1. For ticketed work, use the available skill or connector for the configured
   tracker to verify the issue key and resolve the target repository. Read only
   the issue context needed for that routing; do not begin the full task brief,
   planning, critique, or code exploration yet. For explicit no-ticket work,
   resolve the repository from the user's task context.
2. Extract the issue key. For no-ticket work, derive `<TASK-SLUG>` and use
   `NO-TICKET-<TASK-SLUG>` as the synthetic helper key. If the user supplied a
   branch, run this verified helper with that exact branch:

   ```bash
   uv run --script "<fast-rocket-skill-dir>/../rocket/scripts/ensure_branch.py" \
     --repo <absolute-repo-path> \
     --ticket-key <ISSUE-KEY-OR-SYNTHETIC-NO-TICKET-KEY> \
     --branch-name <exact-user-supplied-branch> \
     --checkout-mode <resolved-checkout> \
     --base-branch main
   ```

   If the branch was omitted for tracked work, let the helper derive its default
   `aryan-binazir/<ISSUE-KEY>` branch by omitting `--branch-name`:

   ```bash
   uv run --script "<fast-rocket-skill-dir>/../rocket/scripts/ensure_branch.py" \
     --repo <absolute-repo-path> \
     --ticket-key <ISSUE-KEY> \
     --checkout-mode <resolved-checkout> \
     --base-branch main
   ```

   For no-ticket work, always supply the derived branch explicitly so it stays
   `aryan-binazir/<task-slug>` rather than inheriting the synthetic helper key:

   ```bash
   uv run --script "<fast-rocket-skill-dir>/../rocket/scripts/ensure_branch.py" \
     --repo <absolute-repo-path> \
     --ticket-key NO-TICKET-<TASK-SLUG> \
     --branch-name aryan-binazir/<task-slug> \
     --checkout-mode <resolved-checkout> \
     --base-branch main
   ```

   In `worktree` mode, keep the helper's default location for any worktree it
   creates: `~/repos/.worktrees/<repo>/<ticket-key>`. Do not pass
   `--worktree-path` to redirect it. In `branch` mode, the helper uses the
   repository path as the checkout and creates or switches the local branch
   there without creating a linked worktree.
3. Parse the helper's JSON. Require `ok: true`, require `checkout_mode` to match
   the resolved config, require `branch` to exactly equal the supplied branch or
   derived default, and use the returned absolute `checkout_path` as the
   authoritative checkout. Call that expected branch the resolved branch. In
   `worktree` mode, the helper may reuse a current or registered worktree or
   create one from an existing local branch, existing remote branch, or latest
   `origin/main`. If it returns a registered matching worktree outside the
   default location, keep using that returned path; do not move or recreate it.
   In `branch` mode, it reuses, creates, or switches the branch in the repository
   checkout and stops if that branch is checked out elsewhere.
4. Immediately tell the user the checkout mode, resolved branch, and checkout
   path so this run is easy to identify among other open work. Checkout setup is
   Fast Rocket's first state-changing action and must finish before the full
   issue read, task briefing, critique, planning, or code exploration.
5. Stop and ask the user before proceeding if the target checkout is dirty, its
   path collides, `main` is unavailable, branch setup fails, the branch is
   checked out elsewhere in `branch` mode, the returned mode or branch
   mismatches, or the returned checkout is not actually on the resolved branch.
   Do not silently switch or edit another checkout.

This reuses Rocket's verified branch/worktree helper without invoking Rocket
contracts or plan profiles. Review invokes Rocket Review only when configured.

From this point forward, run every inspection, context update, plan critique,
implementation action, validation, commit, push, PR action, and review only from
the helper-returned authoritative git `checkout_path`. When delegating, give the
sub-agent that exact path and require it to work only there.

Now read the complete tracked issue; do not rely on its title alone. For
explicit no-ticket work, use the user's task description as the source of
truth. Read the target repository's instructions, relevant code, tests,
documentation, and git state from that checkout. Make sure the goal, accepted
behavior, boundaries, and validation target are understood.

When repository rules require `_scratch/_context/<ticket-key>.md`, resolve the
key from the supplied issue or, for no-ticket work, the task slug resolved from
the intended branch. Never derive it from the currently checked-out branch.
Keep that file current as plans, assumptions, or decisions change, and delete
stale notes instead of accumulating them.

## 2. Brief, Align, And Clarify

If the invocation includes `grill`, require a resolved `grill` block and read
and follow its configured skill from the authoritative checkout. Stop if the
block or skill is unavailable. The grilling session replaces the normal brief
and limited clarification flow below; do not continue to planning or
implementation until the user confirms shared understanding, and skip the
remainder of this section after that confirmation.

Without `grill`, use the normal flow below. This path assumes the task is
reasonably defined and asks only the alignment or validation questions that
materially affect the work.

Before asking any clarification question, give the user a compact task briefing
based on the tracked issue or no-ticket task description and repository evidence:

- **Problem:** what is currently wrong or missing.
- **Outcome:** what the task intends to make true.
- **Scope and constraints:** the important boundaries, acceptance criteria, and
  repo-native constraints that shape the likely implementation.

Then ask one explicit alignment question: whether this is the right direction
or anything should be corrected before proceeding. Do not continue until the
user confirms the direction or provides a correction. Incorporate corrections
and re-inspect affected evidence when needed. This alignment gate does not count
against the clarification-question limit below.

After alignment, continue autonomously through the rest of Fast Rocket. Pause
again only for the material decisions and blockers already required by this
workflow; do not turn the implementation plan into another approval gate.

Ask only questions whose answers could materially change scope, acceptance
criteria, user-facing behavior, API or data contracts, the public test seam, or
hard-to-reverse architecture.

- Ask one question at a time, with a default maximum of three questions.
- Do not ask anything that repository inspection can answer.
- State reversible implementation assumptions and proceed with them.
- Include confirmation of the proposed public test seam required by the `tdd`
  skill; fold it into another material question when practical.
- If the task is clear, proceed immediately apart from any seam confirmation
  still required by `tdd`.
- If material ambiguity remains after three questions, say the task is not
  implementation-ready and ask whether to continue clarifying or proceed with
  explicit assumptions.

Stop for a user decision whenever the answer is hard to undo or would change
user-facing behavior or scope.

## 3. Plan And Get Configured Critique

Write a concise implementation plan covering the intended behavior, affected
areas, confirmed test seams, red-green slices, and required verification.

Use the resolved `critic` runner and its exact non-interactive conventions to
critique the plan against the resolved task, repository evidence, and repo-local
instructions. Give the critic the complete task and request concrete gaps,
risks, unnecessary complexity, and simpler repo-native alternatives. Keep the
critic read-only.

Incorporate actionable feedback. Ask the user only when the critique exposes a
material decision; otherwise state any reversible assumption and continue.
This is one critique round unless the run fails or the user asks for more.

## 4. Implement Test-First

Read and follow the available `tdd` skill completely before implementation.

Work in vertical red-green slices through the confirmed public seams: write one
failing behavior test, run it to observe the expected failure, add only enough
production code to pass, then repeat.

Use the system `implementer` sub-agent by default. If the invocation includes
the literal `implementer` modifier and the resolved config has an `implementer`
block, use that configured runner as the code-changing implementer instead. If
the block is absent, fall back to the system `implementer` sub-agent.

Give either implementer the full task, repository, authoritative checkout,
plan, test seam, and repo-instruction context. Require it to work only in that
checkout, follow the TDD workflow and repository instructions, and not commit,
push, or open a PR. The main agent must inspect status and diff after handoff and
owns validation, commits, pushes, PR creation, and review. Keep changes within
the task's scope and stop if implementation reveals a new material ambiguity.

## 5. Verify, Commit, Push, And Open A Draft PR

Run targeted tests plus every typecheck, lint, test, or other validation required
by the repository. Fix relevant failures; report unrelated or pre-existing
failures honestly.

Immediately before committing, require the current branch to exactly match the
resolved branch. Commit according to repo conventions, then push explicitly
to that branch on `origin`, setting its upstream when needed. Verify the
upstream branch is `origin/<resolved-branch>` and its commit matches
local `HEAD`. Fast Rocket delivery always includes a commit and push; do not
leave completed implementation only in the checkout.

Then create or update the PR as a draft with fully explicit, non-interactive
`gh` commands. Pass `--draft`, `--head`, `--title`, and `--body-file` when
creating it. If a PR already exists and is not a draft, convert it with
`gh pr ready --undo <PR-number-or-URL>` before continuing. Do not use `--fill`,
editor prompts, or implicit fork or push behavior. Follow the repository's PR
title, body, ticket-linking, and assignment rules. For no-ticket work, use the
repository's no-ticket convention for the commit and PR title, such as
`type(no-ticket): description`, and omit ticket linking. Confirm the PR remains
a draft, targets the intended base branch, and its head is the resolved, pushed
implementation branch.

## 6. Run Configured Review

If `review.runner` is `rocket-review`, read and follow the `rocket-review` skill
with no explicit profile so that workflow uses its own local default. Supply the
tracked issue or no-ticket task description as its spec source. Do not also run
the verdict loop below.

Otherwise, use the resolved `review` runner and its exact non-interactive
conventions to review the actual PR diff. Supply the full tracked issue or
no-ticket task description, repo path, base and head commits, PR URL, repo
instructions, changed files, and verification results. Tell the reviewer to
remain read-only, list only concrete actionable findings, and end with exactly
one of:

- `APPROVED`
- `APPROVED WITH FIXES`
- `NO ACTIONABLE FEEDBACK`
- `CHANGES REQUESTED`

Define the choices in the reviewer prompt: use `APPROVED` or
`NO ACTIONABLE FEEDBACK` when no fixes are needed; use `APPROVED WITH FIXES`
only for a complete, enumerated fix list that does not need re-review; use
`CHANGES REQUESTED` when the reviewer must inspect the result of the fixes.

Handle the verdict literally:

- `APPROVED` or `NO ACTIONABLE FEEDBACK`: finish.
- `APPROVED WITH FIXES`: apply every listed fix, rerun relevant verification,
  commit and push the fixes to the resolved branch, and confirm its upstream
  matches local `HEAD`. Then finish without requiring another review.
- `CHANGES REQUESTED`: apply the requested fixes, rerun relevant verification,
  commit and push to the resolved branch, confirm its upstream matches local
  `HEAD`, and ask the same configured reviewer to review the new PR diff again.
  Repeat until it returns a terminal verdict.

Do not infer approval from friendly prose or the absence of high-severity
findings. A malformed or missing verdict is not approval; retry once with the
required format, then stop and report the blocker if it remains malformed.

## Completion

Before completion, verify once more that the resolved branch's upstream commit
matches local `HEAD`. Report the checkout mode and path, branch, PR URL, delivered
behavior, commits, verification performed, the configured review result, and any
remaining caveats. Never merge the PR without the user's explicit consent.
