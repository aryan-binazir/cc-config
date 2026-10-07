---
name: eng-ticket
description: Write or review engineering tickets ready for autonomous implementation via $rocket. Use when scoping work, turning a rough idea into a ticket, or tightening or vetting an existing ticket.
---

# Eng Ticket

Write and review tickets with `$rocket` consumability as the bar: an implementation agent moves with minimal clarification and minimal invention. The deliverable is ticket text; tracker updates and implementation live elsewhere.

## Modes

- `Generate`: idea, rough spec, or notes → finished ticket.
- `Review`: hard critique of an existing ticket against its contract and downstream automation.

Ticket types:

- `Implementation` (default): code-producing work for `$rocket`.
- `Spike / ADR`: investigation or decision record, outside `$rocket`; choose it when the deliverable is clearly a design artifact, research outcome, or decision document.

## Repo Context

Inside a repo, ground the ticket in the codebase first: local rules, relevant structure, and existing packages, modules, and patterns. Name real directories, services, packages, APIs, config patterns, and validation commands. Outside a repo, state assumptions plainly.

## Generate Workflow

1. Pick the ticket type; gather repo context.
2. Ask at most one consolidated clarification round, resolving every structural decision: scope, architecture, integration boundaries, rollout or migration, validation. When in doubt, it's structural.
3. Leave cosmetic decisions (naming, formatting, minor defaults, presentation) as `[DECIDE: ...]`.
4. Write the ticket to its contract, scannable: a ticket, not a design doc.

Output the finished ticket directly in markdown; add commentary only when the user asked for analysis first.

## Implementation Ticket Contract

Headings match the spec sections `rocket-review` reviews against. All required except `## Notes`:

```md
# Title

## Goal

## Accepted scope

## Assumptions

## Out of scope

## Validation approach

## Notes
```

`# Title`: imperative and specific; name the concrete surface.

`## Goal`: why it matters now and what it unlocks or fixes; completion legible to an implementation agent.

`## Accepted scope`: what will actually be built — named files, packages, services, endpoints, commands, schemas, or interfaces when known; integration boundaries and ownership when work spans packages or services; decisions already made that shape the implementation.

Good: `Create internal/redis/client.go with a Client wrapper around go-redis/v9, plus config loading in internal/config and a Ping health check used by startup validation.`
Bad: `Set up Redis with standard connection handling.`

`## Assumptions`: behavior the implementer would otherwise invent — inferred defaults, operational expectations, error handling, boundaries. One too risky to assume becomes a clarification question.

`## Out of scope`: named exclusions that hold scope steady through implementation and review.

`## Validation approach`: runnable checks and concrete manual verification, exact commands when known, proving the accepted scope rather than restating it.

Good: `make lint`, `go test ./internal/redis/...`, and a manual `PING` against the local Redis instance all succeed.
Bad: `Tests pass and the package works correctly.`

`## Notes`: brief hints, background, or follow-ons. Notes outgrowing the contract signal an underspecified ticket.

## Spike / ADR Ticket Contract

```md
# Title

## Goal

## Context

## Questions to answer

## Deliverable

## Out of scope
```

- `Goal`: the decision or uncertainty addressed.
- `Context`: why the investigation matters now.
- `Questions to answer`: specific and bounded.
- `Deliverable`: the output artifact.
- `Out of scope`: keeps the spike from becoming stealth implementation.

## Review Workflow

1. Verify the ticket type and every required section.
2. Flag vagueness with a concrete rewrite.
3. Name assumptions an implementer would have to invent, and missing integration boundaries across systems or packages.
4. Test `Out of scope` against review creep and `Validation approach` for runnable verification over generic claims.
5. Suggest a split for an oversized ticket.

Smells:

- goals with the what but no why
- accepted scope that is a task list with no boundaries
- undefined "basic" handling, "sensible defaults", "standard conventions", or "configurable" — demand the named behavior, defaults, convention, or mechanism
- named files or directories with no stated purpose or contents

## Review Output

```md
## Verdict
[Ready for rocket / Needs work]

## Findings
- [section] - [what is vague or missing and how to fix it]

## Suggested fixes
- [concrete rewrite or added bullet]
```

On request, follow the findings with the full rewritten ticket.

## Quality Bar

Another engineer or agent produces roughly the same implementation without a long planning session. While materially different implementations would all satisfy the ticket, keep tightening.
