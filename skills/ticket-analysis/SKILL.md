---
name: ticket-analysis
description: Analyze a ticket or proposed piece of work to find the relevant files, assess whether the work is actually needed, and write an initial review artifact. Use when the user asks for ticket triage, implementation scoping, or an initial assessment of a Jira or Linear ticket.
---

# Ticket Analysis

Produce an initial ticket review that combines codebase research with a critical assessment of whether the requested work is warranted.

## Required Input

You need the ticket title, description, or both. If the user has not provided enough to identify the work, ask for it.

## Output Destination

Write the result both to the terminal and to `_scratch/_context/initial_review-{branch}.md`, replacing `/` with `-` in the branch name.

## Workflow

### Phase 1: File Research

Spawn a `ticket-researcher` sub-agent to identify relevant files and line-level starting points:
- Use an exploration-oriented sub-agent (e.g., `subagent_type: "Explore"`)
- Prefer a cost-effective model like Sonnet for thorough analysis

Prompt template for the research agent:

```
## Ticket

{ticket info from the user}

Identify the relevant files, modules, and line-level locations for this ticket. Include file paths, approximate line numbers, and a brief note on what each location does relative to the ticket.
```

Capture the full output from Phase 1.

### Phase 2: Need Analysis

Spawn a `ticket-assessor` sub-agent to critically evaluate the ticket's necessity:
- Use a general-purpose sub-agent
- Prefer the same cost-effective model

Prompt template for the assessment agent:

```
## Ticket

{ticket info from the user}

## Codebase Research Findings

{Phase 1 output}

Critically evaluate whether this ticket is necessary. Consider: Is the problem real? Is the proposed solution the right approach? What are the risks? What's the likely implementation shape and effort?
```

Phase 2 depends on Phase 1 findings, so they must run sequentially.

### Phase 3: Compile & Output

Combine both phases into the final report.

**Output Actions:**
1. Display the full report to the user
2. Get the branch name via `git branch --show-current`, replace `/` with `-`
3. Create `_scratch/_context/` directory if it doesn't exist
4. Write to `_scratch/_context/initial_review-{branch}.md`
5. Confirm: "Report saved to `_scratch/_context/initial_review-{branch}.md`"

## Report Structure

```markdown
# Initial Ticket Review

**Ticket**: {ticket summary}
**Branch**: {current branch}
**Date**: {today}

---

## Part 1: Relevant Files & Locations

{Phase 1 output}

---

## Part 2: Necessity Analysis

{Phase 2 output}

---

## Reminder

Before implementing, align with local naming, structure, error handling, and testing conventions.
```

## Notes

- Both agents use Sonnet model for cost-effective thorough analysis
- Phase 2 depends on Phase 1 findings, so they must run sequentially
- If either phase fails, report partial results and note what failed
