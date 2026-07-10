---
name: fast-rocket
description: >-
  Take a reasonably specified Linear or Jira ticket, ticket URL, markdown spec,
  or raw implementation spec through focused clarification, a Cursor-critiqued
  plan, test-driven implementation, verification, PR creation, and a gated
  Codex review. Use this whenever the user invokes fast-rocket or asks for the
  lighter, lower-friction alternative to rocket-plan for an end-to-end ticket.
---

# Fast Rocket

Use this for a reasonably specified ticket that should move quickly from intake
to a reviewed PR. It is separate from Rocket: do not persist a Rocket contract,
resolve Rocket profiles, invoke `rocket-review`, or wait for explicit approval
of the implementation plan.

Never guess past material ambiguity, skip the required external critiques,
write production code before a driving test, or merge unless the user asks.

## 1. Understand The Work

1. Read the supplied Linear or Jira ticket, ticket URL, markdown spec, or raw
   spec. Resolve tracker data with the relevant available skill or connector.
2. Read the target repository's instructions and inspect the relevant code,
   tests, documentation, and current git state before planning.
3. Make sure the goal, accepted behavior, boundaries, and validation target are
   understood. Do not rely on the ticket title alone.

When repository rules require `_scratch/_context/<ticket-key>.md`, resolve the
key from the supplied ticket or intended branch, not the currently checked-out
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
- If material ambiguity remains after three questions, say the ticket is not
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
to critique the plan against the ticket, repository evidence, and repo-local
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
PR creation, and Codex review. Keep changes within the ticket's scope and stop
if implementation reveals a new material ambiguity.

## 5. Verify And Open The PR

Run targeted tests plus every typecheck, lint, test, or other validation required
by the repository. Fix relevant failures; report unrelated or pre-existing
failures honestly.

Commit and push according to repo conventions, verify the upstream branch
matches local `HEAD`, then create or update the PR with fully explicit,
non-interactive `gh` commands. Provide `--head`, `--title`, and `--body-file` as
appropriate; do not use `--fill`, editor prompts, or implicit fork or push
behavior. Follow the repository's PR title, body, ticket-linking, and assignment
rules. Confirm the PR targets the intended branch and its head
is the pushed implementation branch.

## 6. Require A Codex Verdict

Read and follow:

`/home/ar/repos/cc-config/skills/personal_dev/call-codex/SKILL.md`

Use its exact non-interactive CLI conventions to have Codex review the actual PR
diff. Supply the full ticket or spec, repo path, base and head commits, PR URL,
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
  commit and push the fixes, and confirm upstream freshness. Then finish without
  requiring another Codex review.
- `CHANGES REQUESTED`: apply the requested fixes, rerun relevant verification,
  commit and push, confirm upstream freshness, and ask Codex to review the new
  PR diff again. Repeat until Codex returns a terminal verdict.

Do not infer approval from friendly prose or the absence of high-severity
findings. A malformed or missing verdict is not approval; retry once with the
required format, then stop and report the blocker if it remains malformed.

## Completion

Report the PR URL, delivered behavior, commits, verification performed, Codex's
exact terminal verdict, and any remaining caveats. Do not merge the PR unless
the user explicitly asks.
