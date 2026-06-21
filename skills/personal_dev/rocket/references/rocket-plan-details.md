# Rocket Plan Details

Load only the sections needed for the active phase.

## Contract Template

Use exactly:

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

Use `Goal` for the purpose and expected outcome, `Accepted scope` for work that
will be built, `Assumptions` for inferred behavior, `Out of scope` for deliberate
exclusions, and `Validation approach` for the tests/checks that will drive and
verify the work.

## Clarification Coverage

If the source is Linear, make the clarification message self-contained:
- ticket summary
- proposed contract
- ambiguities or gaps
- what you plan to build if the user approves

Ask hard questions one focused round at a time. Do not soften clarification to
be polite or efficient, do not batch everything into one shallow round, and do
not artificially cap rounds. Cover every branch that can change implementation,
especially:
- missing or weak goal/motivation
- product behavior, unhappy paths, empty states, and edge cases
- integration boundaries and downstream/upstream consumers
- data model, migration, and backfill implications
- failure modes, retries, idempotency, and concurrency
- security, permissions, and auth
- performance and scale
- observability expectations
- expected tests and the standard each test enforces
- rollout, feature flags, reversibility, and migrations
- branch naming or ticket identifier requirements for raw specs
- explicit out-of-scope confirmation

Each follow-up round should only ask about concerns that survived previous
answers. Stop when remaining ambiguity can honestly live in `Assumptions` and
the user has confirmed or corrected the contract.

## Critic Prompt

Run after contract settlement and execution-plan drafting, before user approval.
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

Do not ask the critic to implement. If advice is accepted, revise the plan before
showing it to the user. If advice is rejected, state why in the visible plan.

## Approval Plan Requirements

The execution plan must:
- restate the finalized contract
- give concise implementation steps
- explain why the approach is the simplest repo-idiomatic path and which existing
  patterns/integration points it will use
- list strict test-first cycles in order: failing test, command to run it,
  production code it forces, refactor/validation checkpoint
- tie tests back to contract standards such as errors, compatibility,
  accessibility, security, performance, and repo conventions when relevant
- include validation and commit checkpoints aligned to red-green-refactor cycles
  when practical
- explicitly end with `$rocket-review <review-profile>`

The approval gate happens after critique. Do not run another configured critique
after presenting the plan unless the user explicitly asks.

## Linear Managed Region

After user approval and before implementation, update the Linear ticket
description only inside this managed region:

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
- If both markers exist, replace everything between them, inclusive of markers.
- If markers are missing, append a fresh region to the end.
- If only one marker exists, treat it as missing and append a fresh region.
- Never touch content outside the markers.
- Always emit both markers.
- Do not append duplicate managed regions.

## Contract Persistence

Persist to:

```text
_scratch/_contracts/<branch>.md
```

Use raw branch paths and create parents as needed. Do not flatten slash-separated
branch names except where another artifact explicitly says to. The contract file
is the durable handoff to `rocket-review`; do not rely on session memory.

On rerun, reuse a complete existing contract if it has `Goal`, `Accepted scope`,
`Assumptions`, `Out of scope`, and `Validation approach`, and the new input does
not materially change the spec.

## Implementation Discipline

When the `tdd` skill is unavailable, enforce this loop:

1. Write the next failing test.
2. Run the narrow command and observe the intended failure when practical.
3. Write the minimum production code to pass.
4. Refactor without changing behavior.
5. Run targeted validation, then broader validation required by contract/repo rules.
6. Commit the completed checkpoint.

Exceptions are allowed only when strict red-green-refactor is genuinely
impractical or wasteful for that slice. State the exception in the execution plan
or checkpoint commit message and still use the smallest practical test-first
slice.

If independent workstreams exist, sub-agents may parallelize them, but each
workstream follows the same test-first loop.

Stop during implementation if:
- dirty/unrelated changes cannot be safely separated
- tests expose unresolved spec ambiguity
- new blocking ambiguity appears and guessing would be irresponsible
- required permissions or tooling are missing, including inability to push
