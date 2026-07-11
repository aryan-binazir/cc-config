---
name: fast-rocket
description: >-
  Take a Linear issue plus an optional exact user-supplied branch through
  focused clarification, a Cursor-critiqued plan, test-driven implementation,
  verification, commit and push, PR creation, and a gated Codex review. Use this
  whenever the user invokes fast-rocket or asks for the lighter, lower-friction
  alternative to rocket-plan for an end-to-end Linear issue.
---

# Fast Rocket

Use this for a reasonably specified Linear issue that should move quickly from intake
to a reviewed PR. It is separate from Rocket: do not persist a Rocket contract,
resolve Rocket profiles, invoke `rocket-review`, or wait for explicit approval
of the implementation plan.

Expect one required task input and one optional input:

1. A Linear issue ID or URL.
2. Optionally, the exact branch name to use.

For example:

`$fast-rocket BBA-359`

If the branch is omitted, derive it as `aryan-binazir/<resolved-linear-issue-key>`.
If the user supplies a branch, honor it exactly. Ask only when the Linear issue
ID or URL is missing. Resolve the issue as Linear with the available Linear
skill or connector; do not infer another tracker or support ticketless work.

Never guess past material ambiguity, skip the required external critiques,
write production code before a driving test, or merge unless the user asks.

## 1. Resolve The Linear Issue And Worktree

1. Read the supplied Linear issue with the available Linear skill or connector.
   Do not rely on the title alone.
2. Resolve the target repository from the issue and current task context. Do
   not begin planning or code exploration yet.
3. Extract the Linear issue key. If the user supplied a branch, run this
   verified helper with that exact branch:

   ```bash
   uv run --script /home/ar/repos/cc-config/skills/personal_dev/rocket/scripts/ensure_branch.py \
     --repo <absolute-repo-path> \
     --ticket-key <LINEAR-ISSUE-KEY> \
     --branch-name <exact-user-supplied-branch> \
     --base-branch main
   ```

   If the branch was omitted, let the helper derive its default
   `aryan-binazir/<LINEAR-ISSUE-KEY>` branch by omitting `--branch-name`:

   ```bash
   uv run --script /home/ar/repos/cc-config/skills/personal_dev/rocket/scripts/ensure_branch.py \
     --repo <absolute-repo-path> \
     --ticket-key <LINEAR-ISSUE-KEY> \
     --base-branch main
   ```

4. Parse the helper's JSON. Require `ok: true`, require `branch` to exactly
   equal the supplied branch or derived default, and use the returned absolute
   `worktree_path` as the authoritative checkout. Call that expected branch the
   resolved branch. The helper handles a current worktree already on the branch,
   an existing registered worktree, an existing local branch, an existing remote
   branch, or a new branch and worktree from latest `origin/main`.
5. Stop and ask the user before proceeding if the target worktree is dirty, its
   path collides, `main` is unavailable, branch setup fails, the returned branch
   mismatches, or the returned checkout is not actually on the resolved branch.
   Do not silently switch or edit another checkout.

This reuses only Rocket's verified branch/worktree helper. Fast Rocket remains
separate and does not invoke Rocket contracts, profiles, approvals, or review.

Do not assume or independently continue in the caller's checkout. From this
point forward, run every inspection, context update, plan critique,
implementation action, validation, commit, push, PR action, and review only from
the helper-returned authoritative git `worktree_path`. When delegating, give the
sub-agent that exact path and require it to work only there.

Read the target repository's instructions, relevant code, tests,
documentation, and git state from that worktree. Make sure the goal, accepted
behavior, boundaries, and validation target are understood.

When repository rules require `_scratch/_context/<ticket-key>.md`, resolve the
key from the supplied issue or intended branch, not the currently checked-out
branch. Keep that file current as plans, assumptions, or decisions change, and
delete stale notes instead of accumulating them.

## 2. Run A Bounded Clarification Pass

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
- If material ambiguity remains after three questions, say the issue is not
  implementation-ready and ask whether to continue clarifying or proceed with
  explicit assumptions.

Stop for a user decision whenever the answer is hard to undo or would change
user-facing behavior or scope.

## 3. Plan And Get Cursor Critique

Write a concise implementation plan covering the intended behavior, affected
areas, confirmed test seams, red-green slices, and required verification.

Read and follow:

`/home/ar/repos/cc-config/skills/personal_dev/call-cursor/SKILL.md`

Use its exact non-interactive CLI and model-selection conventions to ask Cursor
to critique the plan against the Linear issue, repository evidence, and repo-local
instructions. Give Cursor the complete task and request concrete gaps, risks,
unnecessary complexity, and simpler repo-native alternatives.

Incorporate actionable feedback. Ask the user only when the critique exposes a
material decision; otherwise state any reversible assumption and continue.
This is one critique round unless the run fails or the user asks for more.

## 4. Implement Test-First

Read and follow the verified canonical TDD skill completely before
implementation:

`/home/ar/repos/skills/skills/engineering/tdd/SKILL.md`

Work in vertical red-green slices through the confirmed public seams: write one
failing behavior test, run it to observe the expected failure, add only enough
production code to pass, then repeat.

Obey all repository instructions, including any requirement to delegate
implementation to an `implementer` sub-agent. When delegation is required, the
implementer changes code only and must not push or open the PR. The main agent
must inspect status and diff after handoff, and owns validation, commits, pushes,
PR creation, and Codex review. Keep changes within the issue's scope and stop
if implementation reveals a new material ambiguity.

## 5. Verify And Open The PR

Run targeted tests plus every typecheck, lint, test, or other validation required
by the repository. Fix relevant failures; report unrelated or pre-existing
failures honestly.

Immediately before committing, require the current branch to exactly match the
resolved branch. Commit according to repo conventions, then push explicitly
to that branch on `origin`, setting its upstream when needed. Verify the
upstream branch is `origin/<resolved-branch>` and its commit matches
local `HEAD`. Then create or update the PR with fully explicit, non-interactive
`gh` commands. Provide `--head`, `--title`, and `--body-file` as
appropriate; do not use `--fill`, editor prompts, or implicit fork or push
behavior. Follow the repository's PR title, body, ticket-linking, and assignment
rules. Confirm the PR targets the intended base branch and its head is the
resolved, pushed implementation branch.

## 6. Require A Codex Verdict

Read and follow:

`/home/ar/repos/cc-config/skills/personal_dev/call-codex/SKILL.md`

Use its exact non-interactive CLI conventions to have Codex review the actual PR
diff. Supply the full Linear issue, repo path, base and head commits, PR URL,
repo instructions, changed files, and verification results. Tell Codex to remain
read-only, list only concrete actionable findings, and end with exactly one of:

- `APPROVED`
- `APPROVED WITH FIXES`
- `NO ACTIONABLE FEEDBACK`
- `CHANGES REQUESTED`

Define the choices in the reviewer prompt: use `APPROVED` or
`NO ACTIONABLE FEEDBACK` when no fixes are needed; use `APPROVED WITH FIXES`
only for a complete, enumerated fix list that does not need re-review; use
`CHANGES REQUESTED` when Codex must inspect the result of the fixes.

Handle the verdict literally:

- `APPROVED` or `NO ACTIONABLE FEEDBACK`: finish.
- `APPROVED WITH FIXES`: apply every listed fix, rerun relevant verification,
  commit and push the fixes to the resolved branch, and confirm its upstream
  matches local `HEAD`. Then finish without requiring another Codex review.
- `CHANGES REQUESTED`: apply the requested fixes, rerun relevant verification,
  commit and push to the resolved branch, confirm its upstream matches local
  `HEAD`, and ask Codex to review the new PR diff again. Repeat until Codex
  returns a terminal verdict.

Do not infer approval from friendly prose or the absence of high-severity
findings. A malformed or missing verdict is not approval; retry once with the
required format, then stop and report the blocker if it remains malformed.

## Completion

Before completion, verify once more that the resolved branch's upstream commit
matches local `HEAD`. Report the branch, worktree path, PR URL, delivered
behavior, commits, verification performed, Codex's exact terminal verdict, and
any remaining caveats. Do not merge the PR unless the user explicitly asks.
