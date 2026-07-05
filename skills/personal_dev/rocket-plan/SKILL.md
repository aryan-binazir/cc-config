---
name: rocket-plan
description: >-
  Take a Linear or Jira ticket, ticket URL, markdown spec file, or raw
  implementation spec from intake
  through coding and into a reviewed PR. Use this when the user wants an
  end-to-end implementation flow: clarify the goal, settle an implementation
  contract, run the configured one-round pre-approval critique, drive
  implementation strictly test-first, push, and hand off in-session to the
  configured rocket-review profile. Optional usage: `rocket-plan <profile>`.
---

# Rocket Plan

Use this for end-to-end implementation work, not ordinary planning or ticket
analysis. The promise is: clarify the goal, settle a durable implementation
contract, get one configured pre-approval critique, wait for visible user
approval, implement test-first, push, and invoke `$rocket-review <review-profile>`
in the same session.

Do not skip preflight, treat the original spec as the contract, guess past
material ambiguity, write production code without a driving test, merge the PR,
or hand review to a new/external session.

## Shared Pipeline

Read and follow the full shared pipeline first:

`/home/ar/repos/cc-config/skills/personal_dev/rocket/references/rocket-plan-core.md`

In that file, `<skill>` is `rocket-plan`. This variant implements inline: skip
the `Implementation Capsule` section and the capsule bullet in
`After Approval`; they apply only to capsule-based variants.

## Loaded Skill Rule

If this full `SKILL.md` body was already injected in the current turn as a
`<skill>` block, treat that as the required complete read of this file. The
core reference above must still be read from disk. If only skill metadata is
present, read this file normally before acting.

## Implementation (Inline)

Execute the approved plan yourself through the test-first cycles and the
`Implementation Standards` section of the core reference. Keep all work in the
resolved worktree.
