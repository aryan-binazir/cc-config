---
name: rocket-plan-headless
description: >-
  Take a Linear ticket, Linear ticket URL, or raw implementation spec from intake
  through coding and into a reviewed PR. Use this when the user wants an
  end-to-end implementation flow: clarify the goal, settle an implementation
  contract, run the configured one-round pre-approval critique, drive
  implementation strictly test-first through a configured headless CLI runner
  (`cursor-agent`, `claude`, or `codex`), push, and hand off in-session to the
  configured rocket-review profile. Optional usage: `rocket-plan-headless
  PROFILE`.
---

# Rocket Plan Headless

Use this for end-to-end implementation work, not ordinary planning or ticket
analysis. The promise is: clarify the goal, settle a durable implementation
contract, get one configured pre-approval critique, wait for visible user
approval, write a compact implementation capsule, run implementation through a
configured headless CLI runner, push, and invoke `$rocket-review <review-profile>`
in the same session.

Do not skip preflight, treat the original spec as the contract, guess past
material ambiguity, write production code without a driving test, merge the PR,
or hand review to a new/external session.

## Loaded Skill Rule

If this full `SKILL.md` body was already injected in the current turn as a
`<skill>` block, treat that as the required complete read. Do not shell out to
re-read the same `SKILL.md`. If only skill metadata is present, read the file
normally before acting.

## Token Discipline

- Delegated preflight is mandatory. Discover the sub-agent/delegation tool before
  preflight if it is not already exposed.
- If delegated preflight cannot run, stop before inline checks with
  `delegated_preflight_unavailable`, unless Ar explicitly approves an inline
  bypass in the current conversation.
- Resolve config with `uv run --script
  /home/ar/repos/cc-config/skills/personal_dev/rocket/scripts/resolve_config.py`;
  do not read the YAML files directly unless that script fails.
- Do not read `rocket-review` or downstream skills during preflight; config gives
  the runner list needed for availability checks.
- Load only the needed section of
  `skills/personal_dev/rocket/references/rocket-plan-details.md` for each phase.
- Treat large raw search output as a planning bug. Use surgical, capped
  exploration until implementation begins.
- Give at most one short preflight-start update and one short result/blocker
  update before ticket intake.

## Config

Run `uv run --script
/home/ar/repos/cc-config/skills/personal_dev/rocket/scripts/resolve_config.py`
before choosing a critic, headless implementer, or review handoff. It reads
`rocket.local.yaml` over `rocket.example.yaml` and returns the selected profiles
as JSON. Do not also read the config files by hand after this succeeds.

Use `rocket-plan-headless <profile>` when provided; otherwise use
`defaults.plan_profile`. Stop if `plan_profiles.<profile>` is missing.

Each plan profile provides `critic.name`, `critic.runner` (`claude`, `codex`, or
`cursor`), optional `critic.model`, optional `critic.timeout_ms` defaulting to
`900000`, optional `headless`, and `review_profile` for `$rocket-review`.

When present, `headless` provides the CLI implementer to run after approval:
`headless.runner` is `cursor-agent`, `claude`, or `codex`, `headless.model` is
the runner-specific model value, `headless.reasoning_effort` is the Codex
`model_reasoning_effort` value, `headless.effort` is the Claude effort value,
and `headless.timeout_ms` defaults to `900000`. If `headless.runner` is missing,
use `codex`.

Critic runner commands:
- `claude`: `claude --dangerously-skip-permissions -p "$PROMPT"`
- `codex`: `codex exec --dangerously-bypass-approvals-and-sandbox "$PROMPT"`
- `cursor`: `cursor-agent -p "$PROMPT"`

Headless implementation commands:
- `cursor-agent`: `cursor-agent -p --yolo --trust --model <model> "$PROMPT"`
- `claude`: `claude --dangerously-skip-permissions -p --model <model> --effort <effort> "$PROMPT"`
- `codex`: `codex exec --dangerously-bypass-approvals-and-sandbox --model <model> -c model_reasoning_effort="<effort>" "$PROMPT" < /dev/null`

When `model` is set, pass the runner's supported `--model <model>` flag. For
`cursor-agent`, use `--model composer-2.5` when no model is configured. For
`claude`, use `--model sonnet` and `--effort high` when not configured. For
`codex`, use `--model gpt-5.5` and `-c model_reasoning_effort="xhigh"` when not
configured. Do not pass Cursor `--yolo` or `-f` for plan critique; use YOLO only
for headless implementation. The configured critique is exactly one external
round unless Ar asks for more in the current conversation.

Model research commands used for these defaults:
- `codex exec --help` confirms `--model`, `-c`, and
  `--dangerously-bypass-approvals-and-sandbox`; `codex debug models` lists
  `gpt-5.5` with `low`, `medium`, `high`, and `xhigh` reasoning levels.
- `claude --help` confirms `--model`, `--effort`, `-p`, and
  `--dangerously-skip-permissions`; it documents aliases such as `opus` and
  `sonnet`.
- `cursor-agent --help` confirms `--model`, `--trust`, and `--yolo`; `--yolo`
  is the alias for forced Run Everything mode, and `cursor-agent --list-models`
  lists `composer-2.5` as current.

## Preflight

Read only the `Delegated Preflight Capsule` section of the details reference,
then spawn exactly one no-fork sub-agent for preflight and source/ticket intake.
Delegation is a context-management optimization only: it must perform the same
checks and enforce the same blockers as inline preflight.

Pass the selected plan profile's `headless.runner` as the optional headless
implementation runner for preflight. If no runner is configured, pass `codex`.
When adapting the shared `Delegated Preflight Capsule`, add this one extra line
to the preflight input:

```text
Headless implementation runner to check: <runner>
```

Also add this flag to the `repo_facts.py` command shown in that capsule:

```bash
--headless-runner <runner>
```

Require the returned `tools` object to include `headless_runner` from the script
JSON. Missing or unavailable headless runners are hard blockers.

The main agent consumes only the returned JSON. Do not paste or summarize the
sub-agent transcript. If no valid JSON is returned, stop with
`delegated_preflight_invalid_output`. Do not retry with another preflight agent.

If the returned JSON identifies a ticket key and the worktree is clean, run the
exact command from `context.branch_setup_command` before codebase exploration.
Do not reconstruct the command from memory. If the command is missing, run
`uv run --script
/home/ar/repos/cc-config/skills/personal_dev/rocket/scripts/ensure_branch.py
--input "<original ticket/spec>"` from the target repo. If branch setup reports
`not_on_main_for_branch_create`, `dirty_worktree`, `ticket_key_required`, or
another failure, stop and ask Ar instead of planning from the wrong branch.

Preflight must cover current-repo rules, intended worktree, git state, GitHub
auth, origin reachability, configured critic/review runner availability,
configured headless implementation runner availability, and Linear/source access
when applicable. Missing auth, missing configured runners, unreachable remotes,
inaccessible tickets, and unsafe dirty changes are hard stops.

If repo rules require `_scratch/_context/<branch>.md`, update it when plans,
assumptions, decisions, implementation status, or review handoff state changes.

## Planning Exploration

Read `Planning Exploration Discipline` before pre-contract codebase exploration.
In short: discover candidate files before reading, prefer `rg -l` and exact
symbols/labels, exclude noisy paths, never dump broad search/file output, avoid
wide `sed` reads before the contract, and delegate genuinely broad discovery
only with a compact summary return.

## Intake And Contract

Accept a Linear ticket ID, Linear URL, or raw spec. If multiple sources are
provided, fetched Linear content is source of truth and raw spec text is
supplemental.

Before contract settlement, read `Contract Template` and `Clarification Coverage`
from the details reference. If the `grill-with-docs` skill is available, use it
for the clarification/grilling phase and treat the inline clarification rules as
fallback. Convert the source into the exact contract shape:
`Goal`, `Accepted scope`, `Assumptions`, `Out of scope`, and `Validation
approach`.

The contract is not settled until `Goal` explains why the work matters. If the
ticket/spec is only a task list, push back and ask for the goal or motivation.
Keep asking focused clarification rounds while material ambiguity remains; stop
when unresolved items can honestly live in `Assumptions` and Ar has confirmed or
corrected the contract.

`Validation approach` must drive strict test-first implementation: list each
failing test/check in order, the behavior it protects, the production change it
forces, and targeted/full validation commands. If an automated test is genuinely
inappropriate, say why and name the manual/static replacement.

## Pre-Approval Critique

After the contract is settled and before presenting the plan:
1. Draft an execution plan from the contract.
2. Read `Critic Prompt` and run exactly one configured critic round.
3. Revise the plan for valid feedback.
4. Stop if unresolved material concerns require Ar's input.

Do not call `update_plan`, present the plan for approval, update Linear, create
or switch branches, persist the contract, or edit files until critique is
complete. If the critic times out after the configured budget, report the
timeout instead of silently skipping critique.

## Approval Gate

After critique is complete, call `update_plan` and present the revised plan.
Stop until Ar explicitly approves.

The visible plan must include the finalized contract, concise execution steps,
why this is the simplest repo-idiomatic path, strict test-first cycles,
validation/commit checkpoints, the configured headless implementation runner,
and `$rocket-review <review-profile>` as the final step.

## After Approval

- Linear sync: if a Linear ticket exists, read `Linear Managed Region` and update
  only the marker-bounded region.
- Branch: if the safe preflight branch step did not already create/switch
  branches, create `aryan-binazir/<ticket-id-or-short-slug>` from a clean `main`
  worktree; if already on a matching feature branch, use it. For raw specs, ask
  once for required ticket/branch naming during clarification.
- Contract file: read `Contract Persistence` and persist the settled contract to
  `_scratch/_contracts/<branch>.md` before code changes. Treat `_scratch` as
  local review state unless Ar asks to commit it.
- Implementation capsule: write a compact implementation capsule next to the
  contract at `_scratch/_contracts/<branch>.implementation.md` before code
  changes. This capsule, not the planning transcript, is the source passed to the
  headless implementer.

## Implementation Capsule

The capsule must be purpose-built for coding, not a generic conversation
summary. Include only:
- repo path, branch, ticket/source identifier, and review profile
- finalized contract
- approved execution plan
- strict test-first cycles in order, with commands
- relevant files/ranges already discovered, if any
- repo-local rules that directly affect implementation, commit, push, or review
- validation commands and commit checkpoints
- explicit instruction that the implementer may edit files and run tests but
  must not push or invoke `$rocket-review`

Do not include critique transcripts, clarification history, raw search output,
or rejected alternatives unless they are needed to avoid a known implementation
mistake.

## Implementation

Run exactly one configured headless implementation process after the
implementation capsule is written. Pass only the capsule path, the repo path, and
this task: implement the capsule through strict test-first cycles. Do not paste
the planning transcript into the implementer prompt.

The headless prompt must instruct the implementer to use the `tdd` skill if
available. Otherwise it must read `Implementation Discipline` and follow that
loop. It may edit files, run tests, and create logical checkpoint commits when
that matches the approved plan. It must not push, invoke `$rocket-review`,
broaden scope beyond the capsule, or ask the main agent to reconstruct planning
context.

The headless implementer must return a compact summary only:
- files changed
- commits created, if any
- tests/validation run and results
- blockers or spec ambiguities
- any intentional deviations from the capsule

Run the headless command from the target repo root with the configured
`headless.timeout_ms`. If the configured runner is unavailable, fails
non-interactive auth, exits non-zero, times out, or cannot be invoked with the
configured model, stop with `headless_implementation_failed` and report the
exact runner, command shape, elapsed time, exit status, and useful output. Do not
fallback to inline implementation or another runner unless Ar explicitly
approves that in the current conversation.

The main agent consumes only the compact return, then inspects git status,
reviews the resulting diff or commits as needed, runs targeted validation, runs
repo-required broader validation, and creates any missing logical checkpoint
commits. Run `make lint` before each commit unless repo rules define another
command.

Fix ordinary code/test failures silently after the headless implementer returns.
Stop only when a failure reveals real spec ambiguity or a required
permission/tooling blocker.

## Review Handoff

When implementation is complete, ensure intended changes are committed, push the
current branch, verify upstream matches local `HEAD`, and invoke
`$rocket-review <review-profile>` in the same Codex session.

Do not reimplement `rocket-review` inline, shell out to a separate
`rocket-review` process, describe review as a new session, or reconstruct the
contract from memory. Point review at `_scratch/_contracts/<branch>.md` as the
preferred spec source.

If push, upstream freshness, or `$rocket-review` fails, stop and report the exact
blocker. Do not silently skip review.
