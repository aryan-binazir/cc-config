---
name: architecture-docs
description: Generate Architecture Decision Records (ADRs) documenting why decisions were made, not just what was decided.
author: Senior Software Architect
version: "1.0"
category: documentation
---

# Architecture Documentation Skill

Create ADRs that capture reasoning behind architectural choices. Focus on the "why", not just the "what".

## Core Principles

1. **Context over Commands**: Document problem space, constraints, and forces
2. **Trade-offs over Perfection**: Explicitly state what was sacrificed
3. **Future-Proofing**: Help future teams know if decision still applies
4. **Searchability**: Consistent structure for quick discovery

## ADR Template

```markdown
# ADR-{number}: {Title in Imperative Form}

**Status**: Proposed | Accepted | Deprecated | Superseded by ADR-XXX
**Date**: {YYYY-MM-DD}
**Decision Makers**: {who}
**Tags**: {searchable, keywords}

## Context and Problem Statement
- What problem are we solving?
- What constraints exist (time, resources, skills, systems)?
- Why is this decision needed now?

## Decision Drivers
{In priority order}
1. {e.g., Team expertise in X}
2. {e.g., Need for sub-100ms response}
3. {e.g., Budget constraint of $X}

## Considered Options

### Option 1: {Name}
**Pros**: {advantages}
**Cons**: {disadvantages}
**Trade-offs**: {gains vs losses}

### Option 2: {Name}
{Same structure}

## Decision
**Chosen**: {Name}
**Rationale**: Why chosen over others, which drivers it satisfies, trade-offs accepted, assumptions made

## Consequences
**Positive**: {what improves}
**Negative**: {what we give up, new problems}
**Neutral**: {changes without clear better/worse}

## Implementation Notes
- Key technical details
- Migration path (if replacing existing)
- Monitoring approach
- Rollback strategy

## Validation
- Success metrics
- Warning signs to revisit
- Review timeline

## References
- {relevant docs, prototypes, related ADRs}
```

## When to Create ADR

**Do**:
- Long-term implications
- Difficult/expensive to reverse
- Multiple viable options exist
- Affects multiple teams
- Explaining same decision repeatedly

**Don't**:
- Trivial/easily reversible decisions
- Only one reasonable option
- Personal preference/style

## Workflow

1. **Gather Context**: Interview stakeholders, understand constraints, timeline
2. **Research Options**: Real pros/cons, second-order effects, effort estimates
3. **Draft**: Start with context, document ALL considered options, be honest about trade-offs
4. **Review**: "Would someone 2 years from now understand why?" Get feedback
5. **Maintain**: Update status if superseded, add retrospective notes

## Key Questions

1. What problem are we really solving?
2. What happens if we do nothing?
3. What are we optimizing for? NOT optimizing for?
4. What assumptions are we making?
5. How will we know if this was right?
6. What would make us revisit this?

## Best Practices

**Do**: Be honest about uncertainty, document rejected options, use concrete numbers, link to evidence

**Don't**: Just document the winner, hide downsides, be vague ("better performance" vs "50% reduction in p99")

## Numbering

Use sequential: ADR-001, ADR-002. Don't reuse numbers for deprecated ADRs.
