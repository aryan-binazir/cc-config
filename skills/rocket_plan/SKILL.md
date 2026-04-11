---
name: rocket_plan
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
- expected validation
- rollout or migration expectations when relevant
- branch naming or ticket identifier needs for raw spec work when repo conventions require them

One follow-up round is allowed only if the first answers create new ambiguity.

After that:
- settle the contract
- state reasonable assumptions in `Assumptions`
- stop asking unless continuing would be irresponsible

Do not begin implementation until the contract is settled.

### Plan mode after contract settlement

**important** Immediately after the contract is settled and before implementation starts, enter plan mode with `/plan`, present the plan back to the user, and then stop. Do not edit files, update Linear, create or switch branches, persist the contract, or begin implementation until the user explicitly approves the plan.

That plan must:
- restate the finalized implementation contract
- give a concise execution plan for the work you are about to do
- include validation and commit checkpoints when relevant
- explicitly include `$rocket_review` as the final step

Do not keep this as an internal-only artifact. The user should be able to see the plan you intend to execute before code changes begin.

### Sync the Linear ticket before implementation

Skip this step if no Linear ticket exists.

After clarification settles the contract and before implementation starts, update the ticket description so it matches what will actually be built.

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

Present this plan to the user in the plan-mode step above, then execute against it.

### Implementation rules

- Write the code.
- Follow repo-local conventions from `CLAUDE.md`, `AGENTS.md`, and nearby rules.
- If a repo-local `CLAUDE.md` exists, read it before coding.
- Keep changes scoped to the contract.
- Commit incrementally at logical checkpoints.
- Run `make lint` before each commit unless local repo rules define a different required validation command.
- Run the tests implied by the contract and repo conventions.
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
