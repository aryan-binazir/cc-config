---
name: rocket-plan
description: Take a Linear ticket, Linear ticket URL, or raw implementation spec from intake through coding and into a reviewed PR. Use this when the user wants Codex to ask one strong clarification round, settle a goal and implementation contract, update the Linear ticket when applicable, carry the work through implementation and push, and then hand off in-session to $rocket_review without further babysitting.
---

# Rocket Plan

Use this skill when the user wants an end-to-end implementation flow, not just planning or ticket analysis.

This skill is strict on purpose:
- It does not skip preflight checks.
- It does not treat the original spec as the implementation contract.
- It does not settle a contract without a clear goal.
- It does not silently guess past unresolved ambiguity.
- It does not stop at code completion. The promise ends at a reviewed PR handoff via `$rocket_review`.

## Accepted Inputs

Accept any of the following:
- a Linear ticket ID such as `BBA-11`
- a full Linear ticket URL
- raw spec text

If the user gives multiple sources, do not reject them. Silently prefer the most structured source in this order:
1. full Linear ticket content fetched from Linear
2. raw spec text from the user

## Phase 0: Preflight

Before any work begins, read local repo rules first:
- `CLAUDE.md`
- `AGENTS.md`
- other nearby agent or workflow rules such as `.cursorrules`

Then verify the environment. Stop immediately and report the exact blocker if any check fails.

Required checks:

```bash
git rev-parse --is-inside-work-tree
command -v gh
gh auth status
command -v claude
git ls-remote --exit-code
```

Additional required checks:
- Confirm the current working directory is the intended repo/worktree.
- If the input is a Linear ticket ID or URL, fetch the full ticket and stop if it is inaccessible.
- Inspect `git status -sb` before implementation. If unrelated dirty changes are present and cannot be safely separated, stop and report that instead of guessing.

Do not proceed with a degraded workflow. Missing auth, missing `claude`, unreachable remotes, or inaccessible Linear tickets are hard stops.

## Phase 1: Spec Intake and Clarification

### Acquire and prioritize the source spec

- If the input is a Linear ticket ID or URL, fetch the full ticket content with the available Linear tooling.
- If the input is raw spec text, use it directly.
- If both exist, prefer the fetched Linear content as the source of truth and treat raw spec text as supplemental context.

If the source of truth is a Linear ticket, assume the user may not have read it recently or at all. Do not make the user infer the plan from a ticket they cannot see.

### Require a clear goal

The contract is not settled until the overall goal is explicit.

If the ticket or spec is only a task list and does not explain why the work matters, push back in the clarification round and ask for the goal or motivation behind the work.

If a Linear ticket exists and its current description does not state the goal clearly:
- capture the agreed goal during clarification
- write that goal back into the ticket description before implementation starts

If no Linear ticket exists, capture the goal in the contract and proceed.

### Normalize into an implementation contract

Do not treat the incoming spec as the implementation plan. Convert it into a concise implementation contract with exactly these headings:

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

Use `Goal` for the overall purpose of the work.
Use `Accepted scope` for the work that will actually be built.
Use `Assumptions` for inferred behavior or missing details you had to supply.
Use `Out of scope` for deliberate exclusions so later review does not expand the work retroactively.
Use `Validation approach` for concrete checks such as tests, lint, or manual verification.

The contract must set an implementation quality bar, not just a feature checklist:
- prefer the simplest repo-idiomatic change that satisfies the goal
- reuse existing local patterns, helpers, abstractions, and integration points before introducing new ones
- avoid broad rewrites, duplicate systems, speculative abstraction, and brittle one-off workarounds unless the contract explicitly justifies them
- include any known codebase conventions or architectural constraints that should shape the implementation

`Validation approach` must be specific enough to drive implementation, not just verify it afterward:
- identify the tests the agent expects to add or update before and during coding
- state the behavior, regression, or quality standard each test is meant to protect
- name the targeted and full validation commands the agent expects to run
- if automated tests are not appropriate, explain why and state the manual or static checks that replace them

### Return the ticket plan to the user

Skip this step if no Linear ticket exists.

Before asking clarification questions, give the user a concise but concrete readout of what was loaded from Linear. The user should be able to understand the proposed work without opening the ticket.

That readout must include:
- a short summary of the ticket in plain language
- the proposed implementation contract, using the same `Goal`, `Accepted scope`, `Assumptions`, `Out of scope`, and `Validation approach` headings
- any ambiguities or gaps that need confirmation

Do not ask the user to restate the ticket. Phrase the clarification round so the user can confirm or correct your read of the ticket and proposed plan.

### Clarification rules

Before implementation, ask one consolidated clarification round like a strong peer engineer preparing to own the work. Be thorough enough that the user can answer once and walk away.

If the source spec came from Linear, the clarification message must be self-contained:
- start with the ticket summary and proposed contract
- then ask for confirmation or corrections
- then ask the consolidated clarification questions
- make it clear what you plan to build if the user simply replies with approval

Cover at least:
- the overall goal or motivation if it is missing or weak
- missing product or behavior details
- integration boundaries
- expected validation, including what tests should be created or updated and what standard those tests should enforce
- rollout or migration expectations when relevant
- branch naming or ticket identifier needs for raw spec work when repo conventions require them

One follow-up round is allowed only if the first answers create new ambiguity.

After that:
- settle the contract
- state reasonable assumptions in `Assumptions`
- stop asking unless continuing would be irresponsible

Do not begin implementation until the contract is settled.

### Pre-Plan Claude critique

After the contract is settled and before entering Plan mode:
1. Draft the execution plan.
2. Run the Claude plan critique below.
3. Revise the plan as needed.
4. Stop if unresolved material concerns remain that require user input.

Do not call `update_plan`, present the plan for approval, update Linear, create or switch branches, persist the contract, or begin implementation until the Claude critique loop is complete.

The drafted plan must:
- restate the finalized implementation contract
- give a concise execution plan for the work you are about to do
- explain why the approach is the simplest repo-idiomatic path and which existing patterns or integration points it will use
- include a test-first validation plan that names the tests to create or update, the behavior or standards they enforce, and when they will be run
- include validation and commit checkpoints when relevant
- explicitly include `$rocket_review` as the final step

Ask Claude for a plan critique before presenting the plan for user approval.

Rules:
- Run Claude after the user clarification round has settled the contract and after you have drafted the execution plan.
- Run the first Claude plan critique before user approval, then revise the plan to address material concerns.
- Run follow-up Claude critique rounds only while there are unresolved material concerns about overengineering, codebase fit, validation, scope, or risky assumptions. Do not loop on style preferences, wording, or non-blocking taste comments.
- Cap plan critique at 3 total Claude rounds unless the user explicitly asks for more. If material concerns remain after the cap, present the unresolved concerns to the user instead of continuing the loop.
- Do not ask Claude to implement anything.
- Ask Claude to review the contract and proposed plan for overengineering, avoidable complexity, missing simpler codebase-native approaches, violations of repo-local conventions, weak test strategy, hidden scope expansion, and risky assumptions.
- Include the repo/worktree path, branch, relevant ticket/spec, contract, proposed execution plan, and validation plan.
- If Claude identifies a clearly better simpler approach, revise the plan before showing it to the user.
- If Claude raises a real ambiguity that changes scope or user-facing behavior, ask the user before proceeding.
- If Claude raises feedback that seems potentially correct but depends on product intent, user preference, risk tolerance, rollout expectations, or another judgment the user can reasonably decide, ask the user before accepting or rejecting it.
- For every follow-up critique after round 1, include a prior-feedback ledger in the prompt:
  - accepted Claude recommendations and how the plan changed
  - rejected Claude recommendations and why you are not willing to accept them
  - unresolved concerns that still need Claude to re-check
- If you intentionally reject Claude's advice, state the reason in the user-visible plan.
- Allow up to the full 15-minute budget for the Claude plan critique: `900000` ms. Do not stop early just because Claude has been quiet for a few minutes. If the critique exceeds the full budget, treat it as a timeout failure and report the blocker instead of silently skipping it.

Use a prompt equivalent to:

```text
You are Claude advising Codex before implementation starts.

Review target:
- Repo/worktree: <absolute path>
- Branch: <branch>
- Ticket/spec: <ticket or raw spec summary>

Implementation contract:
<contract>

Proposed execution plan:
<plan>

Validation plan:
<tests and commands>

Prior Claude feedback ledger, for follow-up rounds only:
- Accepted:
  - <recommendation and plan change>
- Rejected:
  - <recommendation and reason it was not accepted>
- Still unresolved:
  - <concern Claude should re-check>

Give brutally honest planning feedback before code is written.

Focus on:
- Is this more complicated than necessary?
- Is there a simpler existing codebase pattern, helper, abstraction, or integration point to use?
- Does the plan violate repo-local conventions or introduce a parallel system?
- Is the test strategy strong enough to drive development?
- Are any assumptions risky, underspecified, or likely to create rework?

Return:
## Blocking
## Simplifications
## Test Strategy
## Risks
## Verdict

No implementation. No compliments. No padding.
```

### Plan mode after Claude critique

**important** After the Claude critique loop is complete and before implementation starts, go to Plan mode, call `update_plan`, and present the revised plan back to the user for feedback. Stop there until the user explicitly approves the plan. Do not run another Claude plan critique after presenting the plan unless the user explicitly asks for one.

Do not edit files, update Linear, create or switch branches, persist the contract, or begin implementation until the user explicitly approves the revised plan.

Do not keep this as an internal-only artifact. The user should be able to see the plan you intend to execute before code changes begin.

### Sync the Linear ticket before implementation

Skip this step if no Linear ticket exists.

After the user explicitly approves the revised plan and before implementation starts, update the ticket description so it matches what will actually be built.

Use a marker-bounded managed region so replacements are safe and predictable:
- look for `<!-- managed:rocket-start -->` and `<!-- managed:rocket-end -->` in the description
- if both markers exist, replace everything between them (inclusive of markers)
- if markers are missing, append the managed region to the end of the description
- never touch content outside the markers

The managed region has this shape:

```md
<!-- managed:rocket-start -->
---
## Rocket Plan Contract

### Goal
...

### Accepted scope
- ...

### Assumptions
- ...

### Out of scope
- ...

### Validation approach
- ...
<!-- managed:rocket-end -->
```

Rules:
- Always emit both markers when writing the managed region.
- Do not append duplicate managed regions. Replace between markers instead.
- If only one marker is found (orphaned state), treat it as missing and append a fresh managed region. Do not try to repair partial markers.

## Phase 2: Branch and Implementation

### Branch handling

Resolve branch state before implementation:
- If the current branch is `main`, create and check out `aryan-binazir/<ticket-id-or-short-slug>`.
- If already on a feature branch, use it.
- If the user provided raw spec text instead of a ticket, ask for a branch name in the clarification round. If the user does not answer, derive `aryan-binazir/<short-descriptive-slug>` and proceed.
- If local commit or PR conventions require a ticket identifier and raw spec work does not provide one, ask once during clarification. If it remains missing, stop instead of inventing a fake ticket.

### Persist or reuse the contract

As soon as the working branch is known and before code changes begin, write the settled contract to:

```text
_scratch/_contracts/<branch>.md
```

Rules:
- Use the raw branch path, not a flattened filename.
- Create parent directories as needed.
- Example: branch `aryan-binazir/BBA-11` maps to `_scratch/_contracts/aryan-binazir/BBA-11.md`.
- Treat this file as the durable handoff artifact for later review. It must survive session interruption between implementation and review. Do not rely on session memory.
- This contract file is local review state by default. Do not commit `_scratch` artifacts unless the user explicitly asks.

On rerun:
- if the current branch already has `_scratch/_contracts/<branch>.md`
- and that file contains a settled contract with `Goal`, `Accepted scope`, `Assumptions`, `Out of scope`, and `Validation approach`
- and the new user input does not materially change the spec

reuse that contract instead of re-asking clarification questions.

If the existing contract is incomplete or the spec materially changed, rebuild it and overwrite the file.

### Execution plan

Before writing code, build the execution plan from:
- `Goal`
- `Accepted scope`
- `Assumptions`
- `Validation approach`
- logical commit checkpoints

The execution plan must turn the `Validation approach` into concrete test work:
- write or update the tests that describe the intended behavior before implementing the corresponding code when the repo's test setup makes that practical
- use those tests to constrain the implementation and catch regressions, not as an afterthought
- tie tests back to the standards in the contract, such as error handling, compatibility, accessibility, performance, security, or repo-local conventions

Present this plan to the user in the plan-mode step above, then execute against it.

### Implementation rules

- Write the code. If the plan identifies independent workstreams, use sub-agents to parallelize them.
- Follow repo-local conventions from `CLAUDE.md`, `AGENTS.md`, and nearby rules.
- If a repo-local `CLAUDE.md` exists, read it before coding.
- Keep changes scoped to the contract.
- Start each implementation checkpoint by adding or updating the tests identified in the plan. Prefer observing the targeted test fail before implementing the production change when doing so is practical and not wasteful.
- Implement only enough production behavior to satisfy the settled contract, repo-local standards, and the tests created for that checkpoint.
- Commit incrementally at logical checkpoints.
- Run `make lint` before each commit unless local repo rules define a different required validation command.
- Run the targeted tests created or updated for the checkpoint, then the broader tests implied by the contract and repo conventions.
- If lint or tests fail because of ordinary code bugs, fix them silently and continue.
- If a failure exposes genuine spec ambiguity rather than a code bug, stop and ask the user. This is the only acceptable mid-implementation interruption.

### Stop conditions during implementation

Stop and report the exact blocker if:
- the working tree contains unrelated or dirty changes that cannot be safely separated
- tests fail in a way that reveals unresolved spec ambiguity
- a new blocking ambiguity appears that the contract did not cover and guessing would be irresponsible
- required permissions or tooling are missing, such as inability to push

## Phase 3: Review Handoff

When implementation is complete:
1. Ensure all intended changes are committed.
2. Push the current branch.
3. Verify that the upstream branch exists and matches local `HEAD`.
4. Invoke `$rocket_review` as a skill in the same Codex session.

The handoff rules are strict:
- Do not reimplement `rocket_review` inline.
- Do not shell out to a separate `rocket_review` process.
- Do not describe this as starting a new session.
- Do not reconstruct the contract from memory if the file already exists.
- Point `$rocket_review` at `_scratch/_contracts/<branch>.md` as the preferred spec source. This is the highest-priority review contract when it exists.
- You may include the Linear ticket reference or raw spec only as secondary context.

If the final push fails, the upstream branch does not exist, or upstream does not match local `HEAD`, stop and report the blocker instead of invoking `$rocket_review`.

If `$rocket_review` cannot run, stop and report the exact blocker. Do not silently skip the review phase.

## What This Skill Does Not Do

- It does not skip the review phase.
- It does not merge the PR.
- It does not replace repo-local rules.
- It does not keep the contract only in session memory.
- It does not settle the contract without a clear goal.
- It does not silently guess past unresolved ambiguity.
- It does not treat `$rocket_review` as an external session handoff. It is an in-session skill invocation.
