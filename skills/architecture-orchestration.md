---
name: architecture-orchestration
description: Coordinate multi-agent architectural changes with separate concerns for design, implementation, testing, and review.
author: Architectural Coordinator
version: "1.0"
category: architecture
---

# Architecture Orchestration

Coordinate complex architectural work using specialized workflows for design, implementation, testing, and review.

## Agent Roles

| Agent | Responsibility | Outputs |
|-------|---------------|---------|
| **Architecture** | Design, documentation, impact analysis | ADR, migration plan, API contracts |
| **Implementation** | Execute architectural changes | Code changes per ADR |
| **Test** | Verify behavior | Unit/integration/regression tests |
| **Review** | Quality assurance | Issues, recommendations, approval |

## 1. Architecture Agent

**Questions to answer**:
- What problem are we solving?
- What are the trade-offs?
- What breaks? Migration path?
- Performance/security implications?

**Output**: ADR in `docs/architecture/decisions/`

## 2. Implementation Agent

**Must reference**: ADR, component boundaries, interface contracts

**Constraints**:
- Don't deviate from ADR without updating it
- Implement incrementally
- Keep commits atomic per component

## 3. Test Agent

**Test levels**:
- Unit: New component behavior
- Integration: Cross-component contracts
- Regression: Nothing broke
- Performance: No degradation

**Report**: Coverage, failures with root cause, benchmarks

## 4. Review Agent

**Checklist**:
- [ ] Code follows ADR
- [ ] Tests cover success/failure paths
- [ ] Error handling appropriate
- [ ] Documentation updated
- [ ] No security vulnerabilities
- [ ] Performance acceptable
- [ ] Migration/rollback plan clear

## Orchestration Strategies

### Sequential (Single Session)
For tightly coupled changes:
```
Architecture -> Implementation -> Testing -> Review
```

### Parallel (Multiple Sessions)
For independent work:
```
T1: Architecture Agent - Design ADR
T2: Test Agent - Write test cases from requirements
T3: Implementation Agent - Implement to spec (after ADR ready)
T4: Review Agent - Review completed work
```

### Iterative (Feedback Loop)
For complex/uncertain changes:
```
Design -> Prototype -> Review -> Revise -> Implement -> Test -> Final Review
```

## Coordination Points

- Architecture blocks Implementation
- Implementation blocks Testing
- Testing blocks Review
- Review may loop back to any stage

**Artifacts**: ADRs, interface definitions, test specs, review feedback

## Example: Refactor Database Layer

```
Architecture: "Create ADR for migrating to repository pattern"
             -> Outputs ADR-0015-repository-pattern.md

Implementation: "Implement repository pattern per ADR-0015"
             -> Repository classes, updated service layer

Test: "Write tests for new repository layer"
             -> Unit tests, integration tests

Review: "Review implementation against ADR-0015"
             -> review.md with findings
```

## When to Use

**Use for**:
- Significant architectural changes
- Cross-cutting refactors
- New major features
- System-wide optimizations

**Don't use for**:
- Simple bug fixes
- Single-file changes
- Trivial refactors

## Prompting Patterns

**Full orchestration**:
"Orchestrate [change] using architecture, implementation, testing, and review agents"

**Specific agent**:
"Act as the Architecture Agent and design [change]"
"Act as the Review Agent and audit [code]"

## Tips

- Start with architecture - don't code before you design
- Keep agents focused - each has one job
- Document handoffs - ADRs, specs, test plans
- Review everything - even when confident
- Iterate when needed - don't force a bad design forward
