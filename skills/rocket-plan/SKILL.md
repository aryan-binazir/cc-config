---
name: rocket-plan
description: Take a Linear ticket, Linear ticket URL, or raw implementation spec from intake through coding and into a reviewed PR. Use this when the user wants an end-to-end implementation flow: clarify the goal, settle an implementation contract, run the configured one-round pre-approval critique, drive implementation strictly test-first, push, and hand off in-session to the configured rocket-review profile. Optional usage: `rocket-plan <profile>`.
---

# Rocket Plan

Use this skill when the user wants an end-to-end implementation flow, not just planning or ticket analysis.

This skill is strict on purpose:
- It does not skip preflight checks.
- It does not treat the original spec as the implementation contract.
- It does not settle a contract without a clear goal.
- It does not silently guess past unresolved ambiguity.
- It does not soften the clarification phase. It grills the user on every branch where the spec is ambiguous, until the contract is unambiguous.
- It does not write production code without a failing test driving it. Test-first is mandatory.
- It does not stop at code completion. The promise ends at a reviewed PR handoff via `$rocket-review <review-profile>`.

## Rocket Config

Before choosing a plan critic or review handoff, read the rocket config:

1. `skills/rocket/rocket.local.yaml` if it exists.
2. `skills/rocket/rocket.example.yaml` for defaults and for any profile missing from local config.

If the user invokes `rocket-plan <profile>`, use `<profile>` as the plan profile name.
If the user invokes bare `rocket-plan`, use `defaults.plan_profile`.

Stop if the selected plan profile does not exist. Do not infer a profile from a hyphenated tool list.

Each plan profile must provide:
- `critic.name`
- `critic.runner`: one of `claude`, `codex`, or `cursor`
- optional `critic.model`
- optional `critic.timeout_ms`; default `900000`
- `review_profile`, passed to `$rocket-review` after implementation

Runner invocation rules:
- `claude`: `claude --dangerously-skip-permissions -p "$PROMPT"`
- `codex`: `codex exec --dangerously-bypass-approvals-and-sandbox "$PROMPT"`
- `cursor`: `cursor-agent -p "$PROMPT"`

When `model` is set, pass it with the runner's supported `--model <model>` flag.
Do not pass `cursor-agent -f`/`--force` for plan critique. The plan critique is advisory and must be read-only.

The plan critique is exactly one external round. Do not run follow-up critique rounds unless the user explicitly asks in the current conversation.

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
git ls-remote --exit-code
```

Also check the configured critic runner:
- `claude` -> `command -v claude`
- `codex` -> `command -v codex`
- `cursor` -> `command -v cursor-agent`

If the selected `review_profile` exists in config, also check every configured review runner now so the promised review handoff is not doomed later.

Additional required checks:
- Confirm the current working directory is the intended repo/worktree.
- If the input is a Linear ticket ID or URL, fetch the full ticket and stop if it is inaccessible.
- Inspect `git status -sb` before implementation. If unrelated dirty changes are present and cannot be safely separated, stop and report that instead of guessing.

Do not proceed with a degraded workflow. Missing auth, missing configured runners, unreachable remotes, or inaccessible Linear tickets are hard stops.

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

`Validation approach` must be specific enough to drive implementation through strict test-first development, not just verify it afterward:
- list the tests that will drive each piece of production behavior, in the order they will be written
- for each test, state the behavior, regression, or quality standard it is meant to protect and the production code it will force into existence
- name the targeted and full validation commands the agent expects to run, including the command used to watch a single test fail before making it pass
- if automated tests are genuinely not appropriate for a slice of work, explain why explicitly and state the manual or static checks that replace them; this is an exception, not a default

### Return the ticket plan to the user

Skip this step if no Linear ticket exists.

Before asking clarification questions, give the user a concise but concrete readout of what was loaded from Linear. The user should be able to understand the proposed work without opening the ticket.

That readout must include:
- a short summary of the ticket in plain language
- the proposed implementation contract, using the same `Goal`, `Accepted scope`, `Assumptions`, `Out of scope`, and `Validation approach` headings
- any ambiguities or gaps that need confirmation

Do not ask the user to restate the ticket. Phrase the clarification round so the user can confirm or correct your read of the ticket and proposed plan.

### Clarification rules

If the `grill-with-docs` skill is available, use it to run this phase and skip the inline rules below. Otherwise, follow the rules below.

Before implementation, grill the user. Go down every branch where the spec is ambiguous, every decision that has a real cost to get wrong, and every assumption that would change the implementation if flipped. Treat planning as the cheap phase and act like a senior engineer who refuses to start work until they understand it end to end.

Be aggressive about surfacing hard questions, not polite about avoiding them. If a question feels uncomfortable to ask, that is usually the signal that it must be asked. The user has explicitly asked for this — do not pre-soften, batch into a single shallow round, or skip questions to "be efficient."

If the source spec came from Linear, the clarification message must be self-contained:
- start with the ticket summary and proposed contract
- then ask for confirmation or corrections
- then ask the hard questions in focused rounds
- make it clear what you plan to build if the user simply replies with approval

Cover at least, and explicitly probe each branch you can think of inside these:
- the overall goal or motivation if it is missing or weak, including the user or business outcome the work must produce
- missing product or behavior details, including unhappy paths, error states, empty states, and edge cases
- integration boundaries, upstream and downstream consumers, and what breaks if this change ships
- data model, migration, and backfill implications
- failure modes, retries, idempotency, and concurrency expectations
- security, permissions, and auth implications
- performance and scale expectations
- observability: what logs, metrics, or traces this work should emit or rely on
- expected validation, including the specific tests that will be written to drive the work and the standard each test must enforce
- rollout, feature flagging, reversibility, and migration expectations when relevant
- branch naming or ticket identifier needs for raw spec work when repo conventions require them
- explicit out-of-scope confirmation so later review does not retroactively expand the work

Run as many clarification rounds as it takes for the contract to be unambiguous. Do not artificially cap rounds. Each follow-up round must be tighter than the last and must only ask about concerns that survived the previous answers — do not re-litigate settled questions.

Stop the grilling only when:
- every remaining ambiguity is small enough that a stated assumption in `Assumptions` is honest and low risk, and
- the user has confirmed or corrected the proposed contract.

Then:
- settle the contract
- state reasonable assumptions in `Assumptions`
- stop asking unless continuing would be irresponsible

Do not begin implementation until the contract is settled.

### Pre-Approval configured critique

After the contract is settled and before presenting the plan for user approval:
1. Draft the execution plan.
2. Run exactly one configured critic round.
3. Revise the plan as needed.
4. Stop if unresolved material concerns remain that require user input.

Do not call `update_plan`, present the plan for approval, update Linear, create or switch branches, persist the contract, or begin implementation until the configured critique is complete.

The drafted plan must:
- restate the finalized implementation contract
- give a concise execution plan for the work you are about to do
- explain why the approach is the simplest repo-idiomatic path and which existing patterns or integration points it will use
- include a strict test-first validation plan that lists each failing test in the order it will be written, the production change it will force into existence, and the command used to run it; tests-after, all-upfront, or alongside-the-code patterns are not acceptable
- include validation and commit checkpoints aligned to red-green-refactor cycles when practical
- explicitly include `$rocket-review <review-profile>` as the final step

Ask the configured critic for a plan critique before presenting the plan for user approval.

Rules:
- Run the critic after the user clarification round has settled the contract and after you have drafted the execution plan.
- Do not ask the critic to implement anything.
- Ask the critic to review the contract and proposed plan for overengineering, avoidable complexity, missing simpler codebase-native approaches, violations of repo-local conventions, weak test strategy, hidden scope expansion, and risky assumptions.
- Include the repo/worktree path, branch, relevant ticket/spec, contract, proposed execution plan, and validation plan.
- If the critic identifies a clearly better simpler approach, revise the plan before showing it to the user.
- If the critic raises a real ambiguity that changes scope or user-facing behavior, ask the user before proceeding.
- If the critic raises feedback that depends on product intent, user preference, risk tolerance, rollout expectations, or another judgment the user can reasonably decide, ask the user before accepting or rejecting it.
- If you intentionally reject the critic's advice, state the reason in the user-visible plan.
- Allow up to the configured timeout, default `900000` ms. Do not stop early just because the critic has been quiet. If the critique exceeds the full budget, treat it as a timeout failure and report the blocker instead of silently skipping it.

Use a prompt equivalent to:

```text
You are <critic.name> advising the implementing agent before implementation starts.

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

Give brutally honest planning feedback before code is written.

Focus on:
- Is this more complicated than necessary?
- Is there a simpler existing codebase pattern, helper, abstraction, or integration point to use?
- Does the plan violate repo-local conventions or introduce a parallel system?
- Is the plan strictly test-first? Does each production change have a failing test scheduled to be written first, in order, with a defined behavior to protect?
- Are any assumptions risky, underspecified, or likely to create rework?

Return:
## Blocking
## Simplifications
## Test Strategy
## Risks
## Verdict

No implementation. No compliments. No padding.
```

### Planning approval gate after critique

**important** After the configured critique is complete and before implementation starts, call `update_plan` and present the revised plan back to the user for feedback. Stop there until the user explicitly approves the plan. Do not claim that the skill can switch collaboration modes by itself; the requirement is the visible planning approval gate. Do not run another configured critique after presenting the plan unless the user explicitly asks for one.

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
- logical commit checkpoints, each scoped to a single red-green-refactor cycle when practical

The execution plan must turn the `Validation approach` into a concrete, ordered sequence of test-first cycles:
- list each failing test that will be written, in the order it will be written
- state the production change each test will force into existence
- explicitly assert that tests are written first and watched fail before any matching production code is written; do not allow tests-after, full-upfront test specs, or tests written in the same step as the production change
- tie tests back to the standards in the contract, such as error handling, compatibility, accessibility, performance, security, or repo-local conventions

Present this plan to the user in the planning approval gate above, then execute against it.

### Implementation rules

If the `tdd` skill is available, use it to drive the test-first red-green-refactor loop and skip the inline rules below. Otherwise, follow the rules below.

- Write the code test-first. Tests must drive development: write the next failing test, watch it fail, then write the minimum production code to make it pass, then refactor. Move on to the next test.
- Tests are not written after the code. Tests are not all written upfront before any code. Tests are not written alongside the code in the same step. Each unit of production code exists because a failing test demanded it.
- The only acceptable exception is when the repo's test setup makes a strict red-green-refactor loop genuinely impractical or wasteful for that specific change. In that case, state the exception explicitly in the execution plan or in a checkpoint commit message and use the closest approximation: write the test first within the smallest practical slice, even if it covers slightly more than one line of production code.
- If the plan identifies independent workstreams, use sub-agents to parallelize them. Each sub-agent must follow the same test-first loop within its workstream.
- Follow repo-local conventions from `CLAUDE.md`, `AGENTS.md`, and nearby rules.
- If a repo-local `AGENTS.md` exists, read it before coding.
- Keep changes scoped to the contract.
- Implement only enough production behavior to satisfy the settled contract, repo-local standards, and the failing test you just wrote.
- Commit incrementally at logical checkpoints. A checkpoint is at minimum one red-green-refactor cycle, not a batched set of untested changes.
- Run `make lint` before each commit unless local repo rules define a different required validation command.
- Run the targeted tests for the current checkpoint, then the broader tests implied by the contract and repo conventions.
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
4. Invoke `$rocket-review <review-profile>` as a skill in the same Codex session, where `<review-profile>` is the selected plan profile's `review_profile`.

The handoff rules are strict:
- Do not reimplement `rocket-review` inline.
- Do not shell out to a separate `rocket-review` process.
- Do not describe this as starting a new session.
- Do not reconstruct the contract from memory if the file already exists.
- Point `$rocket-review <review-profile>` at `_scratch/_contracts/<branch>.md` as the preferred spec source. This is the highest-priority review contract when it exists.
- You may include the Linear ticket reference or raw spec only as secondary context.

If the final push fails, the upstream branch does not exist, or upstream does not match local `HEAD`, stop and report the blocker instead of invoking `$rocket-review`.

If `$rocket-review <review-profile>` cannot run, stop and report the exact blocker. Do not silently skip the review phase.

## What This Skill Does Not Do

- It does not skip the review phase.
- It does not merge the PR.
- It does not replace repo-local rules.
- It does not keep the contract only in session memory.
- It does not settle the contract without a clear goal.
- It does not silently guess past unresolved ambiguity.
- It does not skip or soften the grilling round to be polite or efficient.
- It does not allow tests-after, all-upfront, or alongside-the-code patterns. Tests drive each production change.
- It does not treat `$rocket-review` as an external session handoff. It is an in-session skill invocation.
