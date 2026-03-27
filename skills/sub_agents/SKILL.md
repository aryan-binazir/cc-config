---
name: sub_agents
description: Orchestrate multiple sub-agents with explicit planning, validation, and integration when the user explicitly asks for delegation or parallel agent work. Use when the user wants sub-agents, multi-agent orchestration, or a decomposed agent workflow for a task.
---

# Sub Agents

Orchestrate sub-agents rigorously and only when multi-agent delegation is actually requested or clearly beneficial.

## Required Input

You need a concrete task request. If the user has not supplied one, ask them to restate the task and desired output.

## Modes

- **default mode**: Higher rigor. Multiple sub-agents (design/review/test/risk). Stronger validation.
- **fast mode**: Minimum overhead. At most 1 sub-agent. Minimal planning. Proceed with explicit assumptions. Use when the user asks for a fast or minimal approach.

## Core Rules

- Prefer correctness over ceremony.
- Ask clarifying questions only when ambiguity would materially affect the plan or create risk.
- Never claim tests or commands ran if they did not. Separate "recommended" from "performed" validations.
- Parallelize only truly independent work; otherwise sequence and feed outputs forward.
- Sub-agents don't talk to users directly; they return `open_questions` for you to resolve.
- Reuse and resume existing sub-agents instead of spawning replacements when correcting work. Resuming preserves the sub-agent's full conversation history, avoiding redundant context and reducing token usage.
- Cap correction loops at two retries per sub-task. After 2 failures, escalate to user.

## Sub-Agent Selection

- `the-architect`: significant design/trade-offs/system impacts
- `tech-learning-coach`: step-by-step teaching
- `context-scribe`: updating context files as deliverable
- Otherwise: spawn general sub-agents with explicit roles (Implementer, Reviewer, Tester, Risk Analyst)

## Workflow

### Phase 0: Triage

1. Identify deliverable: code, investigation, design, docs, or mixed.
2. Identify constraints: time, risk, scope, tooling, repo conventions.
3. Decide agent count: Trivial (0-1), Moderate (2-3), Complex/high-risk (3-5).

### Phase 1: Plan

Produce compact execution plan:
- Sub-tasks (no over-fragmenting), chosen sub-agent for each, dependencies
- Success criteria per sub-task (measurable)
- Validation you will run vs cannot run

Only ask user approval for irreversible actions or major scope expansion.

### Phase 2: Delegate

Spawn sub-agents via Task tool. **Parallelize only when**: no shared mutable state, one output won't change another's approach, and each can be validated independently. Otherwise sequence with prior outputs as context.

#### Sub-Agent Prompt Template

```
Context: [Minimal relevant context]

Task: [1-3 sentence task statement]

Success criteria:
- [ ] ...

Output format - return EXACTLY one fenced JSON block:
```json
{
  "result": "...",
  "confidence": "high|medium|low",
  "assumptions": ["..."],
  "open_questions": ["..."],
  "risks": ["..."]
}
```
```

### Phase 3: Validate & Fix Loop

For each sub-task, mark `PASS` or `FAIL` with concrete evidence (diffs, command outputs, test results). If validation can't be performed, state what's missing.

**If FAIL** (max 2 retries):
1. Diagnose failure mode (wrong output / incomplete / errors / wrong approach).
2. **Resume** the existing sub-agent (do NOT spawn a new one) with:
   - Failure diagnosis
   - Specific correction instructions
   - Any new context discovered during validation
3. Re-validate. After 2 failures: escalate to user with:
   - Best-available partial result
   - Specific blockers preventing completion
   - Options and risks for user decision

### Phase 4: Integrate & Present

Deliver single integrated result:
- 1 line per sub-agent: what it produced + confidence
- Validation summary with evidence
- Remaining assumptions/open questions (if any)
