---
name: rocket-plan
description: >-
  Take a Linear ticket, Linear ticket URL, or raw implementation spec from intake
  through coding and into a reviewed PR. Use this when the user wants an
  end-to-end implementation flow: clarify the goal, settle an implementation
  contract, run the configured one-round pre-approval critique, drive
  implementation strictly test-first, push, and hand off in-session to the
  configured rocket-review profile. Optional usage: `rocket-plan <profile>`.
---

# Rocket Plan

Use this for end-to-end implementation work, not ordinary planning or ticket
analysis. The promise is: clarify the goal, settle a durable implementation
contract, get one configured pre-approval critique, wait for visible user
approval, implement test-first, push, and invoke `$rocket-review <review-profile>`
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
before choosing a critic or review handoff. It reads `rocket.local.yaml` over
`rocket.example.yaml` and returns the selected profiles as JSON. Do not also read
the config files by hand after this succeeds.

Use `rocket-plan <profile>` when provided; otherwise use `defaults.plan_profile`.
Stop if `plan_profiles.<profile>` is missing.

Each plan profile provides `critic.name`, `critic.runner` (`claude`, `codex`, or
`cursor`), optional `critic.model`, optional `critic.timeout_ms` defaulting to
`900000`, and `review_profile` for `$rocket-review`.

Runner commands:
- `claude`: `claude --dangerously-skip-permissions -p "$PROMPT"`
- `codex`: `codex exec --dangerously-bypass-approvals-and-sandbox "$PROMPT"`
- `cursor`: `cursor-agent --print --trust "$PROMPT"`

When `model` is set, pass the runner's supported `--model <model>` flag. Do not
pass Cursor force mode for plan critique. The configured critique is exactly one
external round unless Ar asks for more in the current conversation.

## Preflight

Read only the `Delegated Preflight Capsule` section of the details reference,
then spawn exactly one no-fork sub-agent for preflight and source/ticket intake.
Delegation is a context-management optimization only: it must perform the same
checks and enforce the same blockers as inline preflight.

The main agent consumes only the returned JSON. Do not paste or summarize the
sub-agent transcript. If no valid JSON is returned, stop with
`delegated_preflight_invalid_output`. Do not retry with another preflight agent.

If the returned JSON identifies a ticket key, run the exact command from
`context.branch_setup_command` before codebase exploration. Do not reconstruct
the command from memory. This command must fetch the latest `origin/main`; new
ticket worktrees must live under `~/repos/.worktrees/<repo-name>/<ticket-key>`,
and the caller does not need to be on `main`. Parse its JSON output and continue
all exploration, context, contract, implementation, validation, commit, push,
and review work from the returned `worktree_path`. If the command is missing, run
`uv run --script
/home/ar/repos/cc-config/skills/personal_dev/rocket/scripts/ensure_branch.py
--input "<original ticket/spec>"` from the target repo. If branch setup reports
`main_unavailable`, `dirty_target_worktree`, `worktree_path_exists`,
`ticket_key_required`, or another failure, stop and ask Ar instead of planning
from the wrong worktree.

Preflight must cover current-repo rules, intended worktree, git state, GitHub
auth, `origin/main` reachability, configured critic/review runner availability,
and Linear/source access when applicable. Missing auth, missing configured
runners, unreachable `origin/main`, inaccessible tickets, and unsafe dirty
target worktrees are hard stops.

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

Do not call `update_plan`, present the plan for approval, update Linear,
persist the contract, or edit files until critique is complete. The required
preflight branch/worktree setup is the only branch setup allowed before
critique; do not create or switch any other branches. If the critic times out
after the configured budget, report the timeout instead of silently skipping
critique.

## Approval Gate

After critique is complete, call `update_plan` and present the revised plan.
Stop until Ar explicitly approves.

The visible plan must include the finalized contract, concise execution steps,
why this is the simplest repo-idiomatic path, strict test-first cycles,
validation/commit checkpoints, and `$rocket-review <review-profile>` as the
final step.

## After Approval

- Linear sync: if a Linear ticket exists, read `Linear Managed Region` and update
  only the marker-bounded region.
- Branch: if the safe preflight branch step did not already return a worktree,
  run `ensure_branch.py` to create or reuse
  `aryan-binazir/<ticket-id-or-short-slug>` in `~/repos/.worktrees/<repo-name>/`
  from the latest `origin/main`; if already on a matching feature branch, use it.
  For raw specs, ask once for required ticket/branch naming during clarification.
- Contract file: read `Contract Persistence` and persist the settled contract to
  `_scratch/_contracts/<branch>.md` before code changes. Treat `_scratch` as
  local review state unless Ar asks to commit it.

## Implementation

Execute the approved plan through test-first cycles. If the `tdd` skill is
available, use it. Otherwise read `Implementation Discipline` and follow that
loop. Keep changes scoped to the contract and repo-local rules. Commit logical
checkpoints, run targeted validation, run repo-required broader validation, and
run `make lint` before each commit unless repo rules define another command.

Fix ordinary code/test failures silently. Stop only when a failure reveals real
spec ambiguity or a required permission/tooling blocker.

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
