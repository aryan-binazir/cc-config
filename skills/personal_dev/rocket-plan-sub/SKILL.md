---
name: rocket-plan-sub
description: >-
  Take a Linear or Jira ticket, ticket URL, markdown spec file, or raw
  implementation spec from intake
  through coding and into a reviewed PR. Use this when the user wants an
  end-to-end implementation flow: clarify the goal, settle an implementation
  contract, run the configured one-round pre-approval critique, drive
  implementation strictly test-first through a fresh implementation sub-agent,
  push, and hand off in-session to the configured rocket-review profile.
  Optional usage: `rocket-plan-sub PROFILE`.
---

# Rocket Plan Sub

Use this for end-to-end implementation work, not ordinary planning or ticket
analysis. The promise is: clarify the goal, settle a durable implementation
contract, get one configured pre-approval critique, wait for visible user
approval, write a compact implementation capsule, delegate implementation to a
fresh sub-agent, push, and invoke `$rocket-review <review-profile>` in the same
session.

Do not skip preflight, treat the original spec as the contract, guess past
material ambiguity, write production code without a driving test, merge the PR,
or hand review to a new/external session.

## Shared Pipeline

Resolve this `SKILL.md` to its real path first, then resolve `../rocket` relative
to its directory and call that absolute path `<rocket-dir>`. Read and follow the
full shared pipeline first:

`<rocket-dir>/references/rocket-plan-core.md`

In that file, `<skill>` is `rocket-plan-sub`. This is a capsule-based variant:
the `Implementation Capsule` section and the capsule bullet in `After Approval`
apply.

## Loaded Skill Rule

If this full `SKILL.md` body was already injected in the current turn as a
`<skill>` block, treat that as the required complete read of this file. The
core reference above must still be read from disk. If only skill metadata is
present, read this file normally before acting.

## Implementation (Fresh Sub-Agent)

Spawn exactly one fresh implementation sub-agent after the implementation
capsule is written. Pass only the capsule path, the repo path, and this task:
implement the capsule through strict test-first cycles. Do not paste the
planning transcript into the implementer prompt.

The implementer follows the `Implementation Standards` section of the core
reference. It may edit files, run tests, and create logical checkpoint commits
when that matches the approved plan. It must not push, invoke `$rocket-review`,
broaden scope beyond the capsule, or ask the main agent to reconstruct planning
context.

The implementer must return a compact summary only:
- files changed
- commits created, if any
- tests/validation run and results
- blockers or spec ambiguities
- any intentional deviations from the capsule

The main agent consumes only that compact return, then inspects git status,
reviews the resulting diff or commits as needed, runs targeted validation, runs
repo-required broader validation, and creates any missing logical checkpoint
commits following the `Implementation Standards` lint rule.

If a fresh implementation sub-agent cannot run, stop with
`implementation_delegation_unavailable` unless Ar explicitly approves an inline
bypass in the current conversation.
