---
name: architecture-orchestration
description: Coordinate multi-agent architectural changes with separate concerns for design, implementation, testing, and review. Use when making significant architectural changes, refactoring systems, or implementing complex features that need coordinated workflows.
author: Architectural Coordinator
version: "1.0"
category: architecture
---

# Architecture Orchestration

## Overview
For complex architectural work, coordinate multiple specialized workflows that can run in parallel or sequence. Each "agent" handles a specific concern while maintaining architectural coherence.

## Orchestration Pattern

When handling architectural changes:

1. **Architecture Agent** - Design and document
2. **Implementation Agent** - Code changes
3. **Test Agent** - Verify behavior
4. **Review Agent** - Quality assurance

These can run in separate Claude Code sessions or sequentially in one session.

## 1. Architecture Agent

**Responsibility:** Design decisions, documentation, impact analysis

**Workflow:**
```bash
# Create/update ADR
mkdir -p docs/architecture/decisions
# Document:
# - Context: Why this change?
# - Decision: What are we doing?
# - Consequences: What changes?
# - Alternatives: What did we reject?
```

**Outputs:**
- ADR (Architecture Decision Record)
- Migration plan
- Affected components list
- API contracts/interfaces

**Questions to answer:**
- What problem are we solving?
- What are the trade-offs?
- What breaks? What's the migration path?
- What are the performance/security implications?

## 2. Implementation Agent

**Responsibility:** Execute the architectural changes

**Workflow:**
```bash
# Follow the architecture plan
# Make changes to affected components
# Update interfaces/APIs
# Refactor as needed
```

**Must reference:**
- ADR from Architecture Agent
- Component boundaries
- Interface contracts

**Constraints:**
- Don't deviate from ADR without updating it
- Implement incrementally
- Keep commits atomic per component

## 3. Test Agent

**Responsibility:** Verify the implementation

**Workflow:**
```bash
# Unit tests for new behavior
# Integration tests for component interactions
# Regression tests for existing behavior
# Performance tests if relevant
```

**Test levels:**
- Unit: New component behavior
- Integration: Cross-component contracts
- Regression: Nothing broke
- Performance: No degradation

**Reports:**
- Test coverage of new code
- Failed tests with root cause
- Performance benchmarks

## 4. Review Agent

**Responsibility:** Quality assurance and consistency

**Checklist:**
- [ ] Code follows ADR
- [ ] Tests cover success/failure paths
- [ ] Error handling is appropriate
- [ ] Documentation is updated
- [ ] No security vulnerabilities introduced
- [ ] Performance is acceptable
- [ ] Migration path is clear
- [ ] Rollback plan exists

**Outputs:**
- Issues found
- Recommendations
- Approval/block decision

## Orchestration Strategies

### Sequential (Single Session)
Use when changes are tightly coupled:
```
Architecture → Implementation → Testing → Review
```

Ask Claude: "Walk through the architecture orchestration process for [change]"

### Parallel (Multiple Sessions)
Use for independent work:
```
Terminal 1: Architecture Agent - Design ADR
Terminal 2: Test Agent - Write test cases from requirements
Terminal 3: Implementation Agent - Implement to spec (once ADR ready)
Terminal 4: Review Agent - Review completed work
```

### Iterative (Feedback Loop)
Use for complex/uncertain changes:
```
1. Architecture: Initial design
2. Implementation: Prototype core
3. Review: Identify issues
4. Architecture: Revise design
5. Implementation: Refine
6. Test: Verify
7. Review: Final check
```

## Multi-Agent Communication

**Artifacts shared between agents:**
- ADR documents (in `docs/architecture/decisions/`)
- Interface definitions (in code)
- Test specifications (in test files)
- Review feedback (in PR or review.md)

**Coordination points:**
- Architecture blocks Implementation
- Implementation blocks Testing
- Testing blocks Review
- Review may loop back to any stage

## Examples

### Example 1: Refactor Database Layer

**Architecture Agent:**
```bash
# Terminal 1 or Step 1
"Create an ADR for migrating from direct SQL to repository pattern"
# Outputs: ADR-0015-repository-pattern.md
```

**Implementation Agent:**
```bash
# Terminal 2 or Step 2
"Implement the repository pattern per ADR-0015"
# Outputs: repository classes, updated service layer
```

**Test Agent:**
```bash
# Terminal 3 or Step 3
"Write tests for the new repository layer"
# Outputs: test files, integration tests
```

**Review Agent:**
```bash
# Terminal 4 or Step 4
"Review the repository pattern implementation against ADR-0015"
# Outputs: review.md with findings
```

### Example 2: Add New API Endpoint

**Single session orchestration:**
```
"Orchestrate adding a new /api/analytics endpoint:
1. Architecture: Design the endpoint contract and data flow
2. Implementation: Build the endpoint
3. Testing: Write unit and integration tests
4. Review: Check against our API standards"
```

### Example 3: Performance Optimization

**Parallel with feedback:**
```
Architecture Agent: "Analyze performance bottleneck in data processing"
Test Agent: "Create benchmark suite for data processing"
Implementation Agent: "Optimize based on architecture analysis"
Review Agent: "Verify optimization meets performance targets"
```

## When to Use This Skill

✅ Use for:
- Significant architectural changes
- Cross-cutting refactors
- New major features
- System-wide optimizations
- Complex debugging requiring multiple approaches

❌ Don't use for:
- Simple bug fixes
- Single-file changes
- Documentation updates
- Trivial refactors

## Prompting Patterns

**Full orchestration:**
"Orchestrate [architectural change] using architecture, implementation, testing, and review agents"

**Specific agent:**
"Act as the Architecture Agent and design [change]"
"Act as the Test Agent and verify [implementation]"
"Act as the Review Agent and audit [code]"

**Parallel coordination:**
"I'll run Architecture Agent here. In another terminal, I want Implementation Agent to wait for the ADR, then proceed"

## Tips

- **Start with architecture** - Don't code before you design
- **Keep agents focused** - Each has one job
- **Document handoffs** - ADRs, specs, test plans
- **Review everything** - Even if you're confident
- **Iterate when needed** - Don't force a bad design forward
