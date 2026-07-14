---
name: rocket-plan-headless
description: >-
  Take a Linear or Jira ticket, ticket URL, markdown spec file, or raw
  implementation spec from intake
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

## Shared Pipeline

Resolve this `SKILL.md` to its real path first, then resolve `../rocket` relative
to its directory and call that absolute path `<rocket-dir>`. Read and follow the
full shared pipeline first:

`<rocket-dir>/references/rocket-plan-core.md`

In that file, `<skill>` is `rocket-plan-headless`. This is a capsule-based
variant: the `Implementation Capsule` section and the capsule bullet in
`After Approval` apply. The sections below extend the core's `Config`,
`Preflight`, and `Approval Gate` phases with headless-specific requirements.

## Loaded Skill Rule

If this full `SKILL.md` body was already injected in the current turn as a
`<skill>` block, treat that as the required complete read of this file. The
core reference above must still be read from disk. If only skill metadata is
present, read this file normally before acting.

## Headless Config

In addition to the critic fields described in the core reference, the selected
plan profile must provide a `headless` block: `headless.runner` (`cursor`,
`cursor-agent`, `claude`, or `codex`), optional `headless.model`, optional
`headless.reasoning_effort` (Codex `model_reasoning_effort`), optional
`headless.effort` (Claude effort), and `headless.timeout_ms` defaulting to
`900000`.

Model and effort defaults live only in `rocket.example.yaml` /
`rocket.local.yaml`, never in this skill. If the resolved profile has no
`headless.runner`, stop with `headless_config_missing` and ask Ar; do not fall
back to a hardcoded runner or model. When `model`, `reasoning_effort`, or
`effort` is not configured, omit that flag and let the CLI use its own default.

Headless implementation commands:
- `cursor` / `cursor-agent`: `cursor-agent --print --force --trust --model <model> "$PROMPT"`
- `claude`: `claude --dangerously-skip-permissions -p --model <model> --effort <effort> "$PROMPT"`
- `codex`: `codex exec --dangerously-bypass-approvals-and-sandbox --model <model> -c model_reasoning_effort="<effort>" "$PROMPT" < /dev/null`

Use force mode only for headless implementation, never for plan critique.

## Headless Preflight Additions

Pass the selected plan profile's `headless.runner` as the headless
implementation runner for preflight. When adapting the shared
`Delegated Preflight Capsule`, add this one extra line to the preflight input:

```text
Headless implementation runner to check: <runner>
```

Also add this flag to the `repo_facts.py` command shown in that capsule:

```bash
--headless-runner <runner>
```

Require the returned `tools` object to include `headless_runner` from the script
JSON. Missing or unavailable headless runners are hard blockers.

## Approval Gate Addition

The visible plan must also name the configured headless implementation runner.

## Implementation (Headless Runner)

Run exactly one configured headless implementation process after the
implementation capsule is written. Pass only the capsule path, the repo path, and
this task: implement the capsule through strict test-first cycles. Do not paste
the planning transcript into the implementer prompt.

The headless prompt must instruct the implementer to follow the
`Implementation Standards` section of the core reference (use the `tdd` skill if
available). It may edit files, run tests, and create logical checkpoint commits
when that matches the approved plan. It must not push, invoke `$rocket-review`,
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
commits following the `Implementation Standards` lint rule.
