---
name: rocket
description: >-
  Take a configured Linear or Jira issue or explicit no-ticket task plus an optional exact
  user-supplied branch through
  focused clarification, configured plan critique, test-driven implementation,
  verification, commit and push, and configured review. Use this
  whenever the user invokes rocket, rocket codex, or rocket claude, or asks for the lighter, lower-friction
  alternative to rocket-plan for an end-to-end task.
---

# Rocket

The lightweight end-to-end workflow: a reasonably specified task moves from
intake to a reviewed PR without a persisted Rocket contract or a plan-approval
gate. The local config selects Linear or Jira; when the user explicitly says
there is no ticket, accept a clear task description instead.

Inputs — one required, five optional:

1. Optionally, the literal profile `codex` or `claude` immediately after
   `$rocket`; omitted means the configured default profile.
2. An issue ID or URL for the configured tracker, or an explicit `no ticket`
   task description.
3. Optionally, the exact branch name to use.
4. Optionally, the literal `grill` modifier.
5. Optionally, the literal `hunk-review` modifier.
6. Optionally, the literal `implementer` modifier.

For example:

`$rocket BBA-359`

`$rocket codex BBA-359`

`$rocket implementer BBA-359`

`$rocket claude BBA-359 grill`

`$rocket BBA-359 hunk-review`

`$rocket no ticket: fix stale cache invalidation`

Honor a user-supplied branch exactly. When the branch is omitted, derive
`aryan-binazir/<resolved-issue-key>` for tracked work or
`aryan-binazir/<task-slug>` (a reasonable short kebab-case slug) for explicit
no-ticket work. Unless the user explicitly says there is no ticket, ask for an
issue ID or URL from the configured tracker — the configured tracker only.
Treat `grill`, `hunk-review`, and `implementer` as modifiers.

Hold these throughout: resolve material ambiguity before acting, run every
configured critique, write the driving test before production code, and merge
only on the user's explicit request.

## Config

Before interpreting the task input, resolve `<rocket-skill-dir>` as the
absolute directory containing this `SKILL.md`, then run:

```bash
uv run --script "<rocket-skill-dir>/scripts/resolve_config.py"
```

For `$rocket codex` or `$rocket claude`, pass the literal profile as the
resolver's positional argument.

The resolver merges `rocket.example.yaml` with the ignored
`rocket.local.yaml`, then selects the requested `plan_profiles` entry or
`defaults.plan_profile` when no profile was supplied. Stop on any resolver
failure. Checkout mode, tracker, runner, model, and effort come only from the
resolved `plan_profile.config`: `checkout` (one of `worktree` or `branch`),
`tracker`, `critic`, optional `grill`, `review`, and `review_profile`.

For configured `cursor`, `claude`, or `codex` runners, read the matching
`call-cursor`, `call-claude`, or `call-codex` skill before invocation. Pass the
configured `model`, `effort`, `reasoning_effort`, and `timeout_ms` when present;
omit absent options so the runner uses its own defaults. Stop if a configured
runner or model is unavailable — the configured one is the only acceptable
choice.

Pass runner options using their native flags: Cursor `--model`; Claude
`--model` and `--effort`; Codex `--model` plus
`-c model_reasoning_effort="<reasoning_effort>"`. Treat `timeout_ms` as the
maximum wait for the configured invocation, separate from runner CLI flags.

## 1. Prepare The Configured Checkout First

Checkout setup is Rocket's first state-changing action and completes before the
full issue read, task briefing, critique, planning, or code exploration. The
verified branch helper lives at `<rocket-skill-dir>/scripts/ensure_branch.py`.

1. For ticketed work, use the available skill or connector for the configured
   tracker to verify the issue key and resolve the target repository, reading
   only the issue context needed for that routing — the full brief comes after
   checkout. For explicit no-ticket work, resolve the repository from the
   user's task context.
2. Extract the issue key; for no-ticket work, derive `<TASK-SLUG>` and use
   `NO-TICKET-<TASK-SLUG>` as the synthetic helper key. Run the helper with the
   resolved checkout mode taken literally (`worktree` creates or reuses a
   separate Git worktree; `branch` creates or switches the branch in the
   repository checkout):

   ```bash
   uv run --script "<rocket-skill-dir>/scripts/ensure_branch.py" \
     --repo <absolute-repo-path> \
     --ticket-key <ISSUE-KEY-OR-SYNTHETIC-NO-TICKET-KEY> \
     --branch-name <branch> \
     --checkout-mode <resolved-checkout> \
     --base-branch main
   ```

   `--branch-name` handling:
   - User-supplied branch: pass it exactly.
   - Tracked work with the branch omitted: omit `--branch-name`; the helper
     derives its default `aryan-binazir/<ISSUE-KEY>`.
   - No-ticket work: always pass `aryan-binazir/<task-slug>` explicitly so the
     branch stays clear of the synthetic helper key.

   In `worktree` mode, keep the helper's default location for any worktree it
   creates: `<repo>/_scratch/worktrees/<ticket-key>`. In `branch` mode, the
   helper uses the repository path as the checkout and creates or switches the
   local branch there, stopping if that branch is checked out elsewhere.
3. Parse the helper's JSON. Require `ok: true`, `checkout_mode` matching the
   resolved config, and `branch` exactly equal to the supplied or derived
   branch — call that the resolved branch. The returned absolute
   `checkout_path` is the authoritative checkout. In `worktree` mode the helper
   may reuse a current or registered worktree or create one from an existing
   local branch, existing remote branch, or latest `origin/main`; when it
   returns a registered matching worktree outside the default location, keep
   using that returned path as-is.
4. Immediately tell the user the checkout mode, resolved branch, and checkout
   path so this run is easy to identify among other open work. If the
   invocation includes `hunk-review`, also say without blocking:
   `Hunk Review requested. Please ensure the Hunk TUI is running for this
   checkout: cd <checkout_path> && hunk diff origin/main...HEAD --watch`.
5. Stop and ask the user before proceeding if the target checkout is dirty, its
   path collides, `main` is unavailable, branch setup fails, the branch is
   checked out elsewhere in `branch` mode, the returned mode or branch
   mismatches, or the returned checkout is off the resolved branch.

From this point forward, run every inspection, context update, plan critique,
implementation action, validation, commit, push, PR action, and review only
from the helper-returned authoritative `checkout_path`. When delegating, give
the worker that exact path and require it to work only there.

Now read the complete tracked issue — the full body, beyond its title. For
explicit no-ticket work, the user's task description is the source of truth.
Read the target repository's instructions, relevant code, tests, documentation,
and git state from that checkout until the goal, accepted behavior, boundaries,
and validation target are understood.

When repository rules require `_scratch/_context/<ticket-key>.md`, resolve the
key from the supplied issue or, for no-ticket work, from the task slug of the
intended branch — independent of whatever branch is currently checked out.
Keep that file current as plans, assumptions, or decisions change, and delete
stale notes rather than accumulating them.

## 2. Brief, Align, And Clarify

If the invocation includes `grill`: require a resolved `grill` block, then read
and follow its configured skill from the authoritative checkout (stop if either
is unavailable). The grilling session replaces this section's brief and
clarification flow; continue to planning only after the user confirms shared
understanding, then skip the rest of this section.

Otherwise, give the user a compact task briefing based on the tracked issue or
no-ticket task description and repository evidence:

- **Problem:** what is currently wrong or missing.
- **Outcome:** what the task intends to make true.
- **Scope and constraints:** the important boundaries, acceptance criteria, and
  repo-native constraints that shape the likely implementation.

Then ask one explicit alignment question: is this the right direction, or
should anything be corrected first? Continue only once the user confirms or
corrects the direction, incorporating corrections and re-inspecting affected
evidence as needed. This alignment gate sits outside the
clarification-question limit below.

After alignment, continue autonomously through the rest of Rocket — the
implementation plan needs no separate approval. Pause again only for the
material decisions and blockers this workflow already requires.

Ask only questions whose answers could materially change scope, acceptance
criteria, user-facing behavior, API or data contracts, the public test seam, or
hard-to-reverse architecture:

- One question at a time, with a default maximum of three.
- Answer repository-inspectable questions by inspection.
- State reversible implementation assumptions and proceed with them.
- Include confirmation of the proposed public test seam required by the `tdd`
  skill; fold it into another material question when practical.
- A clear task proceeds immediately, apart from any seam confirmation still
  required by `tdd`.
- If material ambiguity remains after three questions, say the task is short of
  implementation-ready and ask whether to continue clarifying or proceed with
  explicit assumptions.

Stop for a user decision whenever the answer is hard to undo or would change
user-facing behavior or scope.

## 3. Plan And Get Configured Critique

Write a concise implementation plan covering the intended behavior, affected
areas, confirmed test seams, red-green slices, and required verification.

Use the resolved `critic` runner and its exact non-interactive conventions to
critique the plan against the resolved task, repository evidence, and
repo-local instructions. Give the critic the complete task and request concrete
gaps, risks, unnecessary complexity, and simpler repo-native alternatives. Keep
the critic read-only.

Incorporate actionable feedback. Ask the user only when the critique exposes a
material decision; otherwise state any reversible assumption and continue. One
critique round, unless the run fails or the user asks for more.

## 4. Implement Test-First

Read and follow the available `tdd` skill completely before implementation.

Work in vertical red-green slices through the confirmed public seams: write one
failing behavior test, run it to observe the expected failure, add only enough
production code to pass, then repeat.

The main agent implements directly. With the `implementer` modifier, read the
`implementer` and `explorer` skills: delegate file changes through the
implementer workers from the authoritative checkout — prompts carry the plan,
test seam, and repo instructions, and workers stay in that checkout — and send
broad recon to the explore worker, citing its findings file in later prompts. Commits, pushes, PRs, and validation stay with
the main agent; inspect status and diff after each handoff. Below the bar,
rerun with a tighter prompt or a stronger worker. If the skill or its workers
are unavailable, stop and report.

In both modes, follow the plan, test seam, TDD workflow, repository
instructions, and scope; stop when implementation reveals a new material
ambiguity.

## 5. Verify, Commit, And Push

Run targeted tests plus every typecheck, lint, test, or other validation the
repository requires. Fix relevant failures; report unrelated or pre-existing
failures honestly. When the change is best proven against a real database or
service stack, read and follow the `verify-sandbox` skill for an ephemeral,
evidence-backed verification pass.

Immediately before committing, require the current branch to exactly match the
resolved branch. Commit according to repo conventions, then push explicitly to
that branch on `origin`, setting its upstream when needed. Verify the upstream
branch is `origin/<resolved-branch>` and its commit matches local `HEAD`.

Rocket delivery always includes committed changes and a push. Rocket itself
leaves PRs untouched: creation belongs to Rocket Review when configured, and
other review runners require an existing PR.

## 6. Interactive Hunk Review

Runs only when the invocation includes `hunk-review`.

Run `hunk skill path`, then read and follow the skill at the returned path.
Require a live session for the authoritative checkout; inspect what it has
loaded and reload it to `diff origin/main...HEAD` when it is showing anything
else.

Review the diff against the task and seed one small batch of focused agent
comments before handing the session to the user. Each time the user asks to
process their comments, read and account for every current user comment before
patching or committing, because `--watch` may reload automatically; the
conversation is the durable comment ledger. Answer questions, patch agreed
changes, verify, commit and push, and reload the session as needed, repeating
without a fixed round limit.

Continue only when the user explicitly asks to proceed to review, for example
`Rocket Review it now`. Before continuing, account for every user comment and
verify that local `HEAD` matches its upstream branch.

## 7. Run Configured Review

If `review.runner` is `rocket-review`: read and follow the `rocket-review`
skill with the resolved `review_profile.name`, supplying the tracked issue or
no-ticket task description as its spec source. Rocket Review owns PR creation
and resolution, and replaces the verdict loop below.

For any other runner, require an existing PR — stop if none exists. Use the
resolved `review` runner and its exact non-interactive conventions to review
the actual PR diff. Supply the full tracked issue or no-ticket task
description, repo path, base and head commits, PR URL, repo instructions,
changed files, and verification results. Tell the reviewer to remain read-only,
list only concrete actionable findings, and end with exactly one of:

- `APPROVED` or `NO ACTIONABLE FEEDBACK` — no fixes needed
- `APPROVED WITH FIXES` — only for a complete, enumerated fix list that needs
  no re-review
- `CHANGES REQUESTED` — the reviewer must inspect the result of the fixes

Define those choices in the reviewer prompt, and handle the verdict literally:

- `APPROVED` or `NO ACTIONABLE FEEDBACK`: finish.
- `APPROVED WITH FIXES`: apply every listed fix, rerun relevant verification,
  commit and push the fixes to the resolved branch, confirm its upstream
  matches local `HEAD`, then finish.
- `CHANGES REQUESTED`: apply the requested fixes, rerun relevant verification,
  commit and push to the resolved branch, confirm its upstream matches local
  `HEAD`, and ask the same configured reviewer to review the new PR diff again.
  Repeat until it returns a terminal verdict.

Approval is only ever one of the exact tokens — friendly prose and an absence
of high-severity findings are neither. A malformed or missing verdict gets one
retry with the required format; if it stays malformed, stop and report the
blocker.

## Completion

Verify once more that the resolved branch's upstream commit matches local
`HEAD`. Report the checkout mode and path, branch, PR URL, delivered behavior,
commits, verification performed, the configured review result, and any
remaining caveats. Merge only on the user's explicit request.
