---
name: eng-ticket
description: Generate or review engineering tickets that are ready for automated implementation. Use this when the user wants to write a ticket, scope work out, turn a rough idea into an implementation-ready ticket, tighten an existing ticket, prepare work for $rocket-plan, or check whether a ticket is good enough for autonomous execution.
---

# Eng Ticket

Generate and review engineering tickets with `rocket-plan` consumability as the quality bar: tickets that let an implementation agent move with minimal clarification and minimal invention.

The deliverable is the ticket text itself — Linear updates and implementation stay outside this skill.

## Modes

Choose one mode:
- `Generate`: the user has an idea, rough spec, or partial notes and wants a finished ticket.
- `Review`: the user has a ticket and wants a hard critique against the template and downstream automation needs.

Choose one ticket type:
- `Implementation`: code-producing work that maps cleanly into `$rocket-plan`.
- `Spike / ADR`: investigation or decision-record work, outside the `$rocket-plan` pipeline.

Default to `Implementation` unless the deliverable is clearly a design artifact, research outcome, or decision document.

## Repo Context

Inside a repo, ground the ticket in the actual codebase before writing: read `CLAUDE.md`, `AGENTS.md`, and similar local rules; inspect the relevant project structure and the existing packages, modules, and patterns related to the work. Name real directories, services, packages, APIs, config patterns, and validation commands whenever the repo context supports them.

Outside a repo, work from the user's description and state assumptions plainly.

## Generate Workflow

1. Decide `Implementation` or `Spike / ADR`.
2. Gather repo context if needed.
3. Ask at most one consolidated clarification round.
4. Push for resolution on structural decisions.
5. Leave unresolved cosmetic details as `[DECIDE: ...]`.
6. Write the ticket in the required structure.
7. Keep it scannable — a ticket that turns into a long design doc is the wrong artifact.

Structural decisions — scope, architecture, integration boundaries, rollout or migration behavior, validation expectations — get resolved in the clarification round. Cosmetic ones — naming, formatting, minor defaults, presentation — may remain as `[DECIDE: ...]`. When in doubt, treat a decision as structural.

## Implementation Ticket Contract

Every implementation ticket uses these headings (`## Notes` optional, all others required):

```md
# Title

## Goal

## Accepted scope

## Assumptions

## Out of scope

## Validation approach

## Notes
```

### Section Rules

`# Title`
- imperative and specific; name the concrete surface area when possible

`## Goal`
- why the work matters now and what it unlocks or fixes
- make completion legible to an implementation agent

`## Accepted scope`
- the concrete things that will actually be built, using named files, packages, services, endpoints, commands, schemas, or interfaces when known
- when work spans packages or services, state the integration boundaries and ownership split
- include already-made decisions that materially shape the implementation

Good: `Create internal/redis/client.go with a Client wrapper around go-redis/v9, plus config loading in internal/config and a Ping health check used by startup validation.`
Bad: `Set up Redis with standard connection handling.`

`## Assumptions`
- surface behavior the implementer would otherwise have to invent: inferred defaults, operational expectations, error-handling and boundary assumptions
- an assumption that feels too risky is probably a clarification question instead

`## Out of scope`
- explicit, named exclusions that prevent retroactive scope expansion during implementation or review

`## Validation approach`
- runnable checks and concrete manual verification, with exact commands when known
- validation proves the accepted scope rather than restating it

Good: `make lint`, `go test ./internal/redis/...`, and a manual `PING` against the local Redis instance all succeed.
Bad: `Tests pass and the package works correctly.`

`## Notes`
- brief hints, background, or follow-on considerations; if this outgrows the contract, the ticket is underspecified

## Spike / ADR Ticket Contract

```md
# Title

## Goal

## Context

## Questions to answer

## Deliverable

## Out of scope
```

- `Goal`: the decision or uncertainty being addressed
- `Context`: why the investigation matters now
- `Questions to answer`: specific and bounded
- `Deliverable`: the actual output artifact
- `Out of scope`: keeps the spike from becoming stealth implementation work

## Review Workflow

1. Verify the ticket type.
2. Check every required section exists.
3. Flag vagueness with a concrete rewrite suggestion.
4. Call out likely assumptions an implementation agent would have to invent.
5. Call out missing integration boundaries when multiple systems or packages are involved.
6. Check whether `Out of scope` is strong enough to stop review creep.
7. Check whether `Validation approach` contains runnable verification rather than generic claims.
8. If the ticket is oversized, suggest a split.

Smells to hunt:
- goals that explain only what, not why
- accepted scope that is really a task list with no boundary definition
- "basic" handling with no definition
- "sensible defaults" with no named defaults
- "standard conventions" with no identified convention
- "configurable" with no specified mechanism
- named files or directories with no purpose or contents

## Review Output

```md
## Verdict
[Ready for rocket-plan / Needs work]

## Findings
- [section] - [what is vague or missing and how to fix it]

## Suggested fixes
- [concrete rewrite or added bullet]
```

If the user asks for a full rewrite, provide the rewritten ticket after the findings.

## Generation Output

Output the finished ticket directly in markdown; add commentary only when the user asked for analysis first.

Optimize for clean mapping into `$rocket-plan`: `Goal` and `Accepted scope` map directly to their contract sections, `Assumptions` minimizes invention, `Out of scope` protects review from expanding the work, and `Validation approach` tells the implementer how to prove completion.

## Quality Bar

A strong ticket lets another engineer or agent produce roughly the same implementation without a long planning session. If multiple materially different implementations would all satisfy the ticket, keep tightening.
