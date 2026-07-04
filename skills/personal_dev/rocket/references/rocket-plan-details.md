# Rocket Plan Details

Load only the sections needed for the active phase. When extracting sections,
use exact line ranges or a fence-aware method; simple `awk`/`sed` heading
extractors that stop at any `^## ` line can break on fenced markdown examples in
`Contract Template`.

## Delegated Preflight Capsule

Use one sub-agent, without forking full conversation context. This is mandatory
for rocket-plan. If delegation cannot be used, stop as blocked before any inline
preflight checks unless Ar explicitly authorizes an inline bypass in the current
conversation.

Pass only:
- repo/worktree absolute path
- original ticket ID, ticket URL, or raw spec
- selected plan profile
- configured critic runner and review runner names
- absolute path to `skills/personal_dev/rocket/scripts/repo_facts.py`
- the explicit checks and JSON schema below

The sub-agent owns only this bounded fact-gathering task and ticket/source
intake. It must not edit files, create branches, update tickets, or draft the
implementation contract. Delegation must not reduce functionality: the same
checks, blockers, and repo-rule awareness apply as inline preflight.

Use a prompt equivalent to:

```text
Check these repository facts and ticket/source details. Return only the JSON
shape below.

Repo/worktree: <absolute path>
Input: <ticket ID, ticket URL, or raw spec>
Selected plan profile: <profile>
Critic runner to check: <runner>
Review runners to check: <runner list>

First run:
uv run --script <repo_facts.py path> --repo <absolute path> --input <input> --critic-runner <runner> --review-runners <comma-separated runners>

Use that JSON as the source of truth for deterministic repository and tool
facts.

Checks:
- Confirm from the script JSON that this is a git worktree and the path is the
  intended repo.
- Read only the current-repo rule files listed by the script JSON and summarize
  relevant rules briefly.
- Confirm from the script JSON that gh is installed/authenticated, `origin/main`
  is reachable, git status is understood, and configured runner commands are
  installed.
- Treat the current checkout's branch and dirty state as context, not a branch
  setup blocker, because the main agent will run the script-provided command to
  create or reuse a ticket worktree from the latest `origin/main`. Dirty state
  blocks only when it is in the target branch's returned worktree.
- If the script JSON reports any blockers, include those blockers unchanged,
  skip ticket fetching, and return `next_action: "stop_blocked"`.
- If there are no blockers and the script JSON has `source.type_hint: "linear"`,
  or the input is a Linear URL, fetch the Linear ticket and summarize only the
  facts needed to start contract settlement.
- Copy the script JSON's `source.type_hint` into `context.source_type_hint`.
- Copy the script JSON's `context.branch_setup_command` into
  `context.branch_setup_command`. Do not run that command; ticket worktree setup
  belongs to the main agent after this bounded check returns.

Return JSON only, no markdown, no command logs, no prose. Keep it under 1200
tokens. Use null/empty arrays instead of long explanations. Shape:
{
  "ok": true,
  "blockers": [],
  "repo": {
    "path": "...",
    "branch": "...",
    "dirty": false,
    "dirty_summary": [],
    "rules_summary": []
  },
  "tools": {
    "gh": true,
    "origin_reachable": true,
    "critic_runner": {"runner": "...", "available": true},
    "review_runners": [{"runner": "...", "available": true}]
  },
  "ticket": {
    "source_type": "linear|raw|unknown",
    "id": null,
    "title": null,
    "summary": null,
    "priority": null,
    "description_gaps": []
  },
  "context": {
    "ticket_key": null,
    "suggested_context_path": null,
    "suggested_branch": null,
    "branch_setup_command": null,
    "source_type_hint": null
  },
  "next_action": "settle_contract|stop_blocked"
}
```

When composing the sub-agent message, do not mention rocket-plan, skills, or
preflight. Give only the bounded task, inputs, checks, and JSON schema.

The main agent consumes only the JSON. If the sub-agent returns extra prose,
extract the JSON and ignore the rest. If no valid JSON is returned, stop as
blocked with `delegated_preflight_invalid_output`; do not spawn another
preflight agent and do not fall back to inline preflight without explicit current
conversation approval from Ar.

## Planning Exploration Discipline

Early rocket-plan work can burn massive context by running broad repo-wide
searches that print every matching line. Avoid that. Planning needs high-signal
evidence, not raw search dumps.

During pre-contract codebase exploration:
- Separate discovery from reading. First find candidate files with `rg -l`,
  `rg --files | head`, or narrowly scoped filename searches.
- Do not run broad repo-wide `rg -n` searches for generic terms like `table`,
  `component`, `roster`, `page`, `draft`, `state`, or `test` unless the output is
  capped and the query is already scoped to a small directory.
- During pre-contract exploration, cap `rg -n` line-hit output at 40 lines or
  fewer. If 40 hits is not enough, refine the query/path instead of raising the
  cap. Larger caps are acceptable for path-only discovery, not line dumps.
- Do not use uncapped `rg --files | rg ...` pipelines during planning. Add
  `head`, narrow the path, or switch to `rg -l` so the transcript gets candidate
  files, not a file-list dump.
- Do not read multiple large files or broad test files before the contract. Use
  symbol/label searches to find the relevant ranges first.
- For UI tickets, resolve the actual route/component surface before reading
  comparison components or broad tests. If several candidates appear, inspect the
  smallest route stubs or exact label hits first, then read only the selected
  component/test ranges.
- Do not use `sed` to read more than 80 source lines during pre-contract
  exploration unless the file is already known to be small and central. Prefer
  `sed -n '<small-range>' path/to/file` around the exact symbol/line hits.
- Exclude noisy paths by default: `node_modules`, `dist`, `coverage`, generated
  files, lockfiles, build artifacts, and package-manager internals.
- Prefer commands that return file paths or small symbol hits before reading
  source:
  - `rg -l "specific phrase|test id|component name" src tests`
  - `rg -n "specificSymbol|data-testid" path/to/file | head -40`
  - `sed -n '<small-range>' path/to/file`
- Cap exploratory output. If a search would return more than roughly 80 lines,
  refine the query or switch to `rg -l`; do not dump it into the transcript.
- Read narrow line ranges around the likely implementation and tests. Only read a
  whole file when it is small or clearly central.
- If broad discovery is genuinely necessary, run it in a delegated explorer and
  require a compact return: candidate files, why each matters, and the next 1-3
  line ranges to inspect. Do not paste raw broad search output back into the main
  thread.

For vague UI tickets, a good default sequence is:
1. Search for exact visible labels, test ids, route names, or component names.
2. List candidate files.
3. Inspect the smallest relevant component/test ranges.
4. Stop once there is enough evidence to write the contract and validation plan.

## Contract Template

Use exactly:

```md
# Implementation Contract

## Goal
- ...

## Accepted scope
- ...

## Assumptions
- ...

## Out of scope
- ...

## Validation approach
- ...
```

Use `Goal` for the purpose and expected outcome, `Accepted scope` for work that
will be built, `Assumptions` for inferred behavior, `Out of scope` for deliberate
exclusions, and `Validation approach` for the tests/checks that will drive and
verify the work.

## Clarification Coverage

If the `grill-with-docs` skill is available, use it to run this phase and skip
the inline rules below. Otherwise, follow the rules below.

Before implementation, grill the user. Go down every branch where the spec is
ambiguous, every decision that has a real cost to get wrong, and every assumption
that would change the implementation if flipped. Treat planning as the cheap
phase and act like a senior engineer who refuses to start work until the contract
is understood end to end.

Be aggressive about surfacing hard questions, not polite about avoiding them. Do
not pre-soften, batch everything into one shallow round, or skip questions to be
efficient.

If the source is Linear, make the clarification message self-contained:
- ticket summary
- proposed contract
- ambiguities or gaps
- what you plan to build if the user approves

Ask hard questions one focused round at a time. Do not soften clarification to
be polite or efficient, do not batch everything into one shallow round, and do
not artificially cap rounds. Cover every branch that can change implementation,
especially:
- missing or weak goal/motivation
- product behavior, unhappy paths, empty states, and edge cases
- integration boundaries and downstream/upstream consumers
- data model, migration, and backfill implications
- failure modes, retries, idempotency, and concurrency
- security, permissions, and auth
- performance and scale
- observability expectations
- expected tests and the standard each test enforces
- rollout, feature flags, reversibility, and migrations
- branch naming or ticket identifier requirements for raw specs
- explicit out-of-scope confirmation

Each follow-up round should only ask about concerns that survived previous
answers. Stop when remaining ambiguity can honestly live in `Assumptions` and
the user has confirmed or corrected the contract.

## Critic Prompt

Run after contract settlement and execution-plan drafting, before user approval.
Use a prompt equivalent to:

```text
You are <critic.name> advising the implementing agent before implementation starts.

Review target:
- Repo/worktree: <absolute path>
- Branch: <branch>
- Ticket/spec: <ticket or raw spec summary>

Implementation contract:
<contract>

Proposed execution plan:
<plan>

Validation plan:
<tests and commands>

Give brutally honest planning feedback before code is written.

Focus on:
- Is this more complicated than necessary?
- Is there a simpler existing codebase pattern, helper, abstraction, or integration point to use?
- Does the plan violate repo-local conventions or introduce a parallel system?
- Is the plan strictly test-first? Does each production change have a failing test scheduled to be written first, in order, with a defined behavior to protect?
- Are any assumptions risky, underspecified, or likely to create rework?

Return:
## Blocking
## Simplifications
## Test Strategy
## Risks
## Verdict

No implementation. No compliments. No padding.
```

Do not ask the critic to implement. If advice is accepted, revise the plan before
showing it to the user. If advice is rejected, state why in the visible plan.

## Approval Plan Requirements

The execution plan must:
- restate the finalized contract
- give concise implementation steps
- explain why the approach is the simplest repo-idiomatic path and which existing
  patterns/integration points it will use
- list strict test-first cycles in order: failing test, command to run it,
  production code it forces, refactor/validation checkpoint
- tie tests back to contract standards such as errors, compatibility,
  accessibility, security, performance, and repo conventions when relevant
- include validation and commit checkpoints aligned to red-green-refactor cycles
  when practical
- explicitly end with `$rocket-review <review-profile>`

The approval gate happens after critique. Do not run another configured critique
after presenting the plan unless the user explicitly asks.

## Linear Managed Region

After user approval and before implementation, update the Linear ticket
description only inside this managed region:

```md
<!-- managed:rocket-start -->
---
## Rocket Plan Contract

### Goal
...

### Accepted scope
- ...

### Assumptions
- ...

### Out of scope
- ...

### Validation approach
- ...
<!-- managed:rocket-end -->
```

Rules:
- If both markers exist, replace everything between them, inclusive of markers.
- If markers are missing, append a fresh region to the end.
- If only one marker exists, treat it as missing and append a fresh region.
- Never touch content outside the markers.
- Always emit both markers.
- Do not append duplicate managed regions.

## Contract Persistence

Persist to:

```text
_scratch/_contracts/<branch>.md
```

Use raw branch paths and create parents as needed. Do not flatten slash-separated
branch names except where another artifact explicitly says to. The contract file
is the durable handoff to `rocket-review`; do not rely on session memory.

On rerun, reuse a complete existing contract if it has `Goal`, `Accepted scope`,
`Assumptions`, `Out of scope`, and `Validation approach`, and the new input does
not materially change the spec.

## Implementation Discipline

When the `tdd` skill is unavailable, enforce this loop:

1. Write the next failing test.
2. Run the narrow command and observe the intended failure when practical.
3. Write the minimum production code to pass.
4. Refactor without changing behavior.
5. Run targeted validation, then broader validation required by contract/repo rules.
6. Commit the completed checkpoint.

Exceptions are allowed only when strict red-green-refactor is genuinely
impractical or wasteful for that slice. State the exception in the execution plan
or checkpoint commit message and still use the smallest practical test-first
slice.

If independent workstreams exist, sub-agents may parallelize them, but each
workstream follows the same test-first loop.

Stop during implementation if:
- dirty/unrelated changes cannot be safely separated
- tests expose unresolved spec ambiguity
- new blocking ambiguity appears and guessing would be irresponsible
- required permissions or tooling are missing, including inability to push
