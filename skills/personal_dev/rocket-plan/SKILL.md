---
name: rocket-plan
description: >-
  Take a Linear ticket, Linear ticket URL, or raw implementation spec from intake
  through coding and into a reviewed PR. Use this when the user wants an
  end-to-end implementation flow: clarify the goal, settle an implementation
  contract, run the configured one-round pre-approval critique, drive
  implementation strictly test-first, push, and hand off in-session to the
  configured rocket-review profile. Optional usage: `rocket-plan <profile>`.
---

# Rocket Plan

Use this when the user wants an end-to-end implementation flow, not just planning
or ticket analysis.

The promise is: clarify the goal, settle a durable implementation contract, get
one configured pre-approval critique, wait for visible user approval, implement
test-first, push, and invoke `$rocket-review <review-profile>` in the same
session.

This skill does not skip preflight checks, treat the original spec as the
contract, guess past material ambiguity, write production code without a driving
test, merge the PR, or treat review as an external/new-session handoff.

## Config

Read config before choosing a critic or review handoff:

1. `skills/personal_dev/rocket/rocket.local.yaml` if it exists.
2. `skills/personal_dev/rocket/rocket.example.yaml` for defaults and missing profiles.

Use `rocket-plan <profile>` when provided; otherwise use `defaults.plan_profile`.
Stop if the selected `plan_profiles.<profile>` does not exist. Do not infer a
profile from a hyphenated tool list.

Each plan profile provides:
- `critic.name`
- `critic.runner`: `claude`, `codex`, or `cursor`
- optional `critic.model`
- optional `critic.timeout_ms`, default `900000`
- `review_profile`, passed to `$rocket-review`

Runner commands:
- `claude`: `claude --dangerously-skip-permissions -p "$PROMPT"`
- `codex`: `codex exec --dangerously-bypass-approvals-and-sandbox "$PROMPT"`
- `cursor`: `cursor-agent -p "$PROMPT"`

When `model` is set, pass the runner's supported `--model <model>` flag. Do not
pass Cursor `-f` for plan critique; it is advisory and read-only.

The configured critique is exactly one external round unless the user explicitly
asks for more in the current conversation.

## Preflight

Before work begins, read repo-local rules:
- `CLAUDE.md`
- `AGENTS.md`
- nearby workflow rules such as `.cursorrules`

Then verify:

```bash
git rev-parse --is-inside-work-tree
command -v gh
gh auth status
git ls-remote --exit-code
git status -sb
```

Confirm the current working directory is the intended repo/worktree.

Check the configured critic runner with `command -v claude`, `command -v codex`,
or `command -v cursor-agent`. Also check every runner in the selected
`review_profile` so the promised review handoff is not doomed later.

If the input is a Linear ticket ID or URL, fetch the full ticket and stop if it
is inaccessible. If unrelated dirty changes cannot be safely separated, stop
instead of guessing. Missing auth, missing configured runners, unreachable
remotes, and inaccessible tickets are hard stops.

If repo rules require `_scratch/_context/<branch>.md`, update it when plans,
assumptions, decisions, implementation status, or review handoff state changes.

## Intake

Accept:
- a Linear ticket ID
- a full Linear ticket URL
- raw spec text

If multiple sources are provided, prefer fetched Linear ticket content as source
of truth and treat raw spec text as supplemental context.

If the source is Linear, return a concise self-contained readout before asking
clarifying questions: ticket summary, proposed contract, and ambiguities/gaps.
Do not make the user open the ticket or restate it.

## Contract

Do not treat the incoming spec as the plan. Convert it into an implementation
contract with exactly:

```md
# Implementation Contract

## Goal
- ...

## Accepted scope
- ...

## Assumptions
- ...

## Out of scope
- ...

## Validation approach
- ...
```

The contract is not settled until `Goal` explains why the work matters. If the
ticket/spec is only a task list, push back and ask for the goal or motivation.

Set a quality bar in the contract:
- choose the simplest repo-idiomatic path that satisfies the goal
- reuse existing helpers, patterns, abstractions, and integration points
- include repo conventions and architectural constraints that should shape the work
- avoid broad rewrites, duplicate systems, speculative abstractions, and brittle
  one-off workarounds unless the contract justifies them
- make `Out of scope` explicit so review does not expand the work retroactively

`Validation approach` must drive strict test-first implementation: list the
tests in order, what behavior/quality standard each protects, what production
change each forces, and targeted/full validation commands. If an automated test
is genuinely inappropriate, say why and name the manual/static replacement. Tie
tests back to contract standards such as error handling, compatibility,
accessibility, security, performance, and repo conventions when relevant.

Before each phase that depends on exact details, read the relevant section of:

`skills/personal_dev/rocket/references/rocket-plan-details.md`

Use `Contract Template` and `Clarification Coverage` during contract settlement,
`Critic Prompt` before critique, `Linear Managed Region` before ticket sync,
`Contract Persistence` before writing/reusing the contract file, and
`Implementation Discipline` before implementation.

If `grill-with-docs` is available, use it for the clarification phase. Otherwise
ask focused rounds yourself. Keep asking only while material ambiguity remains.
Do not soften clarification to be polite or efficient. Do not batch into one
shallow round or cap rounds artificially; each follow-up should be tighter than
the last. Stop grilling when unresolved items are low-risk enough to record as
assumptions and the user has confirmed or corrected the contract.

## Pre-Approval Critique

After the contract is settled and before presenting the plan:
1. Draft an execution plan from the contract.
2. Run exactly one configured critic round.
3. Revise the plan for valid feedback.
4. Stop if unresolved material concerns require user input.

Do not call `update_plan`, present the plan for approval, update Linear, create
or switch branches, persist the contract, or edit files until the configured
critique is complete.

Ask the critic to review the contract and proposed plan for overengineering,
avoidable complexity, missing repo-native approaches, convention violations,
weak test strategy, hidden scope expansion, and risky assumptions. Include the
repo/worktree path, branch, ticket/spec, contract, execution plan, and validation
plan. Use the exact prompt shape from the details reference.

Allow the configured timeout, default `900000` ms. Do not stop early because the
critic is quiet. If the full budget is exceeded, report the timeout instead of
silently skipping critique.

If the critic identifies a clearly better simpler approach, revise the plan. If
it raises scope/user-facing ambiguity or feedback depending on product intent,
risk tolerance, rollout expectations, or user preference, ask the user before
accepting or rejecting it. If you reject advice, state why in the visible plan.

## Approval Gate

After critique is complete, call `update_plan` and present the revised plan to
the user. Stop there until the user explicitly approves.

The visible plan must include:
- the finalized implementation contract
- concise execution steps
- why this is the simplest repo-idiomatic path
- strict test-first validation cycles, including each failing test in order, the
  production change it forces, and the command used to run it
- validation and commit checkpoints
- `$rocket-review <review-profile>` as the final step

Do not edit files, update Linear, create or switch branches, persist the
contract, or begin implementation before explicit approval.

## Linear Sync

Skip if no Linear ticket exists.

After user approval and before implementation, update the ticket description so
it matches what will actually be built. Use the marker-bounded managed region in
the details reference. Do not touch content outside the markers or append
duplicates.

If the ticket description lacks a clear goal and one was agreed during
clarification, include that goal in the managed contract.

## Branch And Contract File

Resolve branch state after approval:
- If on `main`, create and check out `aryan-binazir/<ticket-id-or-short-slug>`.
- If already on a feature branch, use it.
- For raw specs, ask for a branch name during clarification; if unanswered,
  derive `aryan-binazir/<short-descriptive-slug>`.
- If repo conventions require a ticket ID and raw spec work lacks one, ask once
  during clarification and stop if it remains missing.

Persist the settled contract before code changes:

```text
_scratch/_contracts/<branch>.md
```

Use the raw branch path, not a flattened filename. Example:
`aryan-binazir/BBA-11` maps to `_scratch/_contracts/aryan-binazir/BBA-11.md`.
Treat `_scratch` as local review state and do not commit it unless the user asks.

On rerun, reuse an existing complete contract for the current branch when the new
input does not materially change the spec. If incomplete or changed, rebuild and
overwrite it.

## Implementation

Execute the approved plan through test-first cycles. If the `tdd` skill is
available, use it. Otherwise:
- Write the next failing test first, watch it fail, then add the minimum
  production code, then refactor.
- Do not write tests after code, all upfront, or in the same step as the matching
  production code.
- If strict red-green-refactor is genuinely impractical for a slice, state the
  exception and use the smallest practical test-first approximation.
- Keep changes scoped to the contract and repo-local rules.
- Commit logical checkpoints, preferably one red-green-refactor cycle each.
- Run targeted tests for the current checkpoint, then broader validation implied
  by the contract and repo conventions.
- Run `make lint` before each commit unless repo rules define another command.
- Fix ordinary code/test failures silently; stop only when a failure reveals real
  spec ambiguity or a required permission/tooling blocker.

## Review Handoff

When implementation is complete:
1. Ensure intended changes are committed.
2. Push the current branch.
3. Verify upstream exists and matches local `HEAD`.
4. Invoke `$rocket-review <review-profile>` in the same Codex session.

Do not reimplement `rocket-review` inline, shell out to a separate
`rocket-review` process, describe the handoff as a new session, or reconstruct
the contract from memory. Point review at `_scratch/_contracts/<branch>.md` as
the preferred spec source and include any Linear/raw spec only as secondary
context.

If push, upstream freshness, or `$rocket-review` fails, stop and report the exact
blocker. Do not silently skip the review phase.
