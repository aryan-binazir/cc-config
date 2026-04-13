# Rocket workflow: minimal work-environment delta

## Source inspected
- `skills/rocket_plan/SKILL.md` (home/original orchestrator)
- `skills/rocket_review/SKILL.md` (home/original two-reviewer flow)
- `skills/rocket_review_claude_only/SKILL.md` (Claude-only variant, intentionally **not** the base to adapt)

## Decision: does `rocket_plan` need a `_w` variant?

Short answer: **No**, not for your stated goals.

Reasoning:
- The workflow shape you want (`rocket_plan -> rocket_review -> draft PR`) is already encoded in `rocket_plan` and should remain unchanged.
- The hard work-environment deltas (Claude permissions mode, Codex -> Cursor, timeout/kill behavior, reviewer identity, and likely ticket-system sync adaptation) all live in `rocket_review`.
- Keeping `rocket_plan` unchanged and replacing/installing a work-compatible skill under the same name `rocket_review` is the smallest viable change set.

When `rocket_plan_w` would become necessary:
- Only if you must change ticket intake away from Linear in phase-0/1 plan intake itself.
- Or if branch naming must be enforced from planning stage (`ab-<ticket-id>`) rather than review/implementation conventions.

## Required command substitutions in `rocket_review`

1) Claude round 1 command
- Replace:
  - `claude --dangerously-skip-permissions -p "$PROMPT"`
- With:
  - `claude -p --allowedTools <read-only-tools> "$PROMPT"`

Suggested minimal read-only allowlist for review:
- `Read`
- `Glob`
- `Grep`
- `Bash(git status -sb)`
- `Bash(git diff --name-only origin/<branch>...HEAD)`
- `Bash(git diff origin/<branch>...HEAD)`

2) Round 2 runner
- Replace:
  - `command -v codex`
  - `codex exec "$PROMPT"`
- With:
  - `command -v cursor-agent`
  - `cursor-agent -p --force --output-format text --workspace "<repo-path>" "$PROMPT"`

3) Preflight checks
- Replace Codex availability check with Cursor Agent CLI availability check.

4) Ticket sync tooling
- Replace Linear-specific sync step with Jira sync implementation (or temporarily gate it behind explicit Jira tooling availability check and skip with warning).

## Required prompt text changes

Round 1 (Claude reviewer) prompt identity:
- Replace:
  - `You are Claude reviewing work completed by Codex.`
- With:
  - `You are Claude reviewing work completed by ChatGPT.`

Round 2 (Cursor reviewer) prompt identity:
- Replace Codex persona text with ChatGPT persona text, for example:
  - `You are ChatGPT reviewing work completed on this branch.`

`code_review_parallel` instruction:
- Keep for Claude round 1 if available.
- Remove/inline for Cursor round 2 (Cursor likely does not have your Codex skill inventory).

## Timeout and process-lifecycle changes for Cursor (`-p` hang issue)

Keep the existing 900000 ms budget.

Required behavior:
- Launch Cursor review as a child process.
- Wait until completion or budget exhaustion.
- On budget exhaustion, send termination signal and confirm process death.
- If still alive after grace period, force-kill.
- Classify as timeout only when full budget is consumed.
- If wrapper stops early, classify as premature abort.

Also update parsing assumptions:
- Cursor output (`--output-format text`) may include logs before final answer.
- Extract final structured review block from trailing output.
- Keep existing normalization behavior for priority-based findings if headings are missing.

## Jira / GitHub / auth / workspace assumptions to adjust

1) Ticket sync
- Home skill assumes Linear API/tooling for managed region writeback.
- Work variant must call Jira CLI/API (or skip sync with explicit status if unavailable).

2) PR platform
- Home flow is GitHub + `gh`.
- If work remains GitHub, keep as-is and preserve **draft PR** creation requirement.
- If not GitHub, PR creation/comment commands must be swapped platform-wide (GitLab/Bitbucket/Azure), not just patched in one place.

3) Auth assumptions
- Add `CURSOR_API_KEY` requirement check before round 2.
- Optional one-time setup note:
  - `cursor-agent cli config approvalMode auto`

4) Workspace routing
- Always pass `--workspace <repo-path>` for detached Cursor runs to avoid cwd ambiguity.

## Draft PR behavior to preserve

Enforce draft creation at PR creation call site:
- `gh pr create --draft --head <branch> --title ... --body-file ...`

If existing PR is non-draft and your policy requires draft at this stage, either:
- stop and report mismatch, or
- convert state if tooling supports it and policy allows.

## Additional home-only assumptions found beyond your two known blockers

1) `rocket_plan` has explicit Linear-first intake/update assumptions throughout phases 0/1.
- This is the one thing that can force a `rocket_plan_w` if your intake source is Jira IDs/URLs rather than raw spec.

2) `rocket_plan` branch naming defaults to `aryan-binazir/<...>`.
- That conflicts with required work branch naming `ab-<ticket-id>`.
- If branch naming is enforced in planning, this is a genuine `rocket_plan` delta.

3) `rocket_plan` default validation says `make lint` unless local rules override.
- This is acceptable if your repos rely on `CLAUDE.md` conventions; no required fork if override path is respected.

4) Both skills assume GitHub `gh` CLI for PR/comment flow.
- Works only if work environment remains on GitHub.

## Minimal-diff recommendation

Use this path first:
1. Keep `rocket_plan` unchanged.
2. Implement a work-compatible `rocket_review` under the same skill name.
3. In that work `rocket_review`, do only these substitutions:
   - Claude dangerous flag -> allowedTools
   - Codex binary/exec -> cursor-agent detached exec
   - reviewer identity text updates
   - remove round-2 `code_review_parallel` dependency
   - add robust timeout + kill semantics for Cursor hangs
   - enforce draft PR creation
   - swap Linear sync tail to Jira sync (or explicit skip behavior)

If and only if planning-stage ticket ingestion must be Jira-native and branch naming must always be `ab-<ticket-id>`, then add `rocket_plan_w` as a second step.

## Remaining blockers for office readiness

- Unknown Jira sync contract (exact CLI and managed-region strategy) is unresolved.
- Unknown PR host (if not GitHub) is a hard blocker because current review flow depends on `gh`.
- Unknown Claude `--allowedTools` exact syntax for your installed CLI version may require one validation pass.
- Cursor non-interactive auth (`CURSOR_API_KEY`) and local approval mode must be configured on the runner.
