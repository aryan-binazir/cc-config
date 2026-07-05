---
name: eng-ticket
description: Generate or review engineering tickets that are ready for automated implementation. Use this when the user wants to write a ticket, scope work out, turn a rough idea into an implementation-ready ticket, tighten an existing ticket, prepare work for $rocket-plan, or check whether a ticket is good enough for autonomous execution.
---

# Eng Ticket

Generate and review engineering tickets with `rocket-plan` consumability as the primary quality bar.

The goal is not just readable tickets. The goal is tickets that let an implementation agent move with minimal clarification and minimal invention.

## Modes

Choose one mode:
- `Generate` when the user has an idea, rough spec, or partial notes and wants a finished ticket.
- `Review` when the user already has a ticket and wants a hard critique against the template and downstream automation needs.

Choose one ticket type:
- `Implementation` for code-producing work that should map cleanly into `$rocket-plan`.
- `Spike / ADR` for investigation or decision-record work that does not directly feed into `$rocket-plan`.

Default to `Implementation` unless the deliverable is clearly a design artifact, research outcome, or decision document.

## Repo Context

If you are inside a repo, ground the ticket in the actual codebase before writing:
- read `CLAUDE.md`, `AGENTS.md`, and similar local workflow rules if present
- inspect the relevant project structure
- inspect existing packages, modules, and patterns related to the requested work

This should change the ticket from abstract to concrete. Prefer naming real directories, services, packages, APIs, config patterns, and validation commands when the repo context supports them.

If you are not inside a repo, work from the user's description only and say assumptions plainly.

## Generate Workflow

1. Decide whether the ticket is `Implementation` or `Spike / ADR`.
2. If needed, gather repo context first.
3. Ask at most one consolidated clarification round.
4. Push for resolution on structural decisions that affect:
   - scope
   - architecture
   - integration boundaries
   - rollout or migration behavior
   - validation expectations
5. Allow unresolved cosmetic details to remain as `[DECIDE: ...]`.
6. Write the ticket in the required structure.
7. Keep it scannable. If the ticket turns into a long design doc, it is the wrong artifact.

Use `[DECIDE: ...]` only for low-impact unresolved choices such as naming, minor defaults, or presentation details. Do not use `[DECIDE: ...]` as a substitute for missing scope or architecture.

## Implementation Ticket Contract

Every implementation ticket must use these headings:

```md
# Title

## Goal

## Accepted scope

## Assumptions

## Out of scope

## Validation approach

## Notes
```

`## Notes` is optional. All other sections are required.

### Section Rules

`# Title`
- imperative and specific
- name the concrete surface area when possible

`## Goal`
- explain why the work matters now
- explain what it unlocks or fixes
- make completion legible to an implementation agent

`## Accepted scope`
- list the concrete things that will actually be built
- use named files, packages, services, endpoints, commands, schemas, or interfaces when known
- when work spans multiple packages or services, explicitly state the integration boundaries and ownership split
- include already-made decisions directly here when they materially shape the implementation

Good: `Create internal/redis/client.go with a Client wrapper around go-redis/v9, plus config loading in internal/config and a Ping health check used by startup validation.`
Bad: `Set up Redis with standard connection handling.`

`## Assumptions`
- surface behavior the implementer would otherwise have to invent
- include inferred defaults, operational expectations, error-handling assumptions, and boundary assumptions
- if an assumption feels too risky, it is probably a clarification question instead

`## Out of scope`
- be explicit and concrete
- prevent retroactive scope expansion during implementation or review
- prefer named exclusions over vague phrases like "advanced features"

`## Validation approach`
- include runnable checks and concrete manual verification
- prefer exact commands when known
- validation should prove the accepted scope, not restate it

Good: `make lint`, `go test ./internal/redis/...`, and a manual `PING` against the local Redis instance all succeed.
Bad: `Tests pass and the package works correctly.`

`## Notes`
- keep brief
- use for hints, background, or follow-on considerations that do not belong in scope
- if this becomes longer than the contract itself, the ticket is underspecified

### Structural vs Cosmetic Decisions

Treat unresolved decisions in two buckets:

- Structural: affects scope, architecture, migration, integration, or validation. Push to resolve these in the clarification round.
- Cosmetic: affects naming, formatting, minor defaults, or other low-risk details. These may remain as `[DECIDE: ...]`.

When in doubt, treat the decision as structural.

## Spike / ADR Ticket Contract

Use this shape for non-implementation investigation tickets:

```md
# Title

## Goal

## Context

## Questions to answer

## Deliverable

## Out of scope
```

Rules:
- `Goal` explains the decision or uncertainty being addressed
- `Context` explains why the investigation matters now
- `Questions to answer` must be specific and bounded
- `Deliverable` names the actual output artifact
- `Out of scope` prevents the spike from becoming stealth implementation work

## Review Workflow

When reviewing a ticket:

1. Verify the ticket type.
2. Check every required section exists.
3. Flag vagueness with a concrete rewrite suggestion.
4. Call out likely assumptions an implementation agent would have to invent.
5. Call out missing integration boundaries when multiple systems or packages are involved.
6. Check whether `Out of scope` is strong enough to stop review creep.
7. Check whether `Validation approach` contains runnable verification rather than generic claims.
8. If the ticket is oversized, suggest a split.
9. Be direct. No fluff.

Pay special attention to these smells:
- goals that explain only what, not why
- accepted scope that is really a task list with no boundary definition
- "basic" handling with no definition
- "sensible defaults" with no named defaults
- "standard conventions" with no identified convention
- "configurable" with no specified mechanism
- named files or directories with no purpose or contents

## Review Output

For review mode, prefer this structure:

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

For generation mode, output the finished ticket directly in markdown. Do not wrap it in commentary unless the user asked for analysis first.

Optimize for clean mapping into `$rocket-plan`:
- `Goal` should map directly to `Goal`
- `Accepted scope` should map directly to `Accepted scope`
- `Assumptions` should minimize what `rocket-plan` has to invent
- `Out of scope` should protect review from expanding the work
- `Validation approach` should tell the implementer how to prove completion

## What This Skill Does Not Do

- It does not create or update Linear tickets directly.
- It does not implement code.
- It does not replace `$rocket-plan` clarification entirely.
- It does not defer structural ambiguity by normalizing everything into `[DECIDE: ...]`.

## Output Quality Bar

A strong ticket should let another engineer or agent produce roughly the same implementation without a long planning session.

If you can already see multiple materially different implementations that would all satisfy the ticket, the ticket is still too vague.
