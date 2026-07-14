---
name: rocket-review
description: >-
  Run the final configured review loop for a completed branch, whether or not a
  PR already exists. Use this whenever the user says `rocket-review`, asks for
  the final review loop, or wants Codex to ensure the current branch has a PR,
  run the configured reviewers, patch what should be patched, and post one final
  PR summary comment. Optional usage: `rocket-review PROFILE`.
---

# Rocket Review

Use this after implementation is complete enough for external review.

Take the current checked-out branch, ensure it has a PR, run the configured
review profile against the supplied spec, patch what should be patched, keep a
strict local diary, and post exactly one final PR summary comment.

This skill does not define implementation work, alter reviewer-reported severity
or verdict tokens, merge the PR, run retired CodeRabbit workflows, or silently
switch review profiles.

## Config

Run `uv run --script
/home/ar/repos/cc-config/skills/personal_dev/rocket/scripts/resolve_config.py`
before choosing reviewers. It reads `rocket.local.yaml` over
`rocket.example.yaml`; do not also read the config files by hand after this
succeeds.

Use `rocket-review <profile>` when provided; otherwise use `defaults.review_profile`.
Stop if the selected `review_profiles.<profile>` does not exist. Do not infer a
profile from a hyphenated tool list.

Each review profile provides `slash_command`, `summary_title`, `diary_name`, and
ordered `reviewers`. Each reviewer provides `name`, `runner`, optional `model`,
optional Codex `reasoning_effort`, and `max_rounds`.

Runner commands:
- `claude`: `claude --dangerously-skip-permissions -p "$PROMPT"`
- `codex`: `codex exec --dangerously-bypass-approvals-and-sandbox "$PROMPT" < /dev/null`
- `cursor`: `cursor-agent --print --trust "$PROMPT"`

When `model` is set, pass the runner's supported `--model <model>` flag.
When a Codex reviewer sets `reasoning_effort`, pass
`-c model_reasoning_effort="<reasoning_effort>"`. Stop if
`reasoning_effort` is configured for a non-Codex reviewer instead of silently
ignoring it.
Reviewers are read-only: do not pass Cursor force mode, and the reviewer prompt
must forbid file modification. Patching findings is the main agent's job.

## Preflight

Before PR resolution or reviewer round 1:

```bash
git rev-parse --is-inside-work-tree
git branch --show-current
command -v gh
gh auth status
git status -sb
```

Also check each configured runner with `command -v claude`, `command -v codex`,
or `command -v cursor-agent`.

Read repo-local rules before generating a PR title or body:
- `CLAUDE.md`
- `AGENTS.md`
- nearby workflow rules such as `.cursorrules`

Stop if the repo/worktree, branch, `gh`, auth, runner availability, or runner
non-interactive auth is not ready.

If repo rules require `_scratch/_context/<branch>.md`, update it when review
plans, assumptions, decisions, fixes, or final review state change.

## Branch State

Reviewers must review the actual pushed branch state.

Before round 1:
- If local changes are review-ready and belong on this branch, commit them using
  repo conventions and push before invoking reviewers.
- If local changes are unrelated, ambiguous, or not ready, stop and ask.
- If there is no upstream branch yet, push before PR creation.

After every push, verify the upstream branch exists and local `HEAD` matches it.
Stop if upstream is stale or missing.

Between rounds:
- If you patch findings, make one follow-up commit for that round and push it.
- Do not amend unless the user asks.
- Do not create bookkeeping commits.
- Do not rerun the same reviewer against unchanged `HEAD`; record unresolved
  findings and move on after that round.

## Spec Source

Supply the spec directly to every reviewer. Do not make reviewers discover it.

Use the best available source in this order:
1. `_scratch/_contracts/<branch>.md` from `rocket-plan`
2. a Linear or Jira ticket ID (resolve the tracker with available tooling; do
   not assume it from the key format)
3. a full Linear or Jira ticket URL
4. a markdown spec file path supplied by Ar
5. explicit fallback spec text

Use the raw branch path for contracts. Example:
`aryan-binazir/BBA-11` maps to `_scratch/_contracts/aryan-binazir/BBA-11.md`.
Treat `_scratch` review artifacts as local state; do not commit them unless the
user explicitly asks.

When a local contract exists, pass its absolute path to the reviewer and include
or summarize its contents in the prompt. It must include the review target:
`Goal`, `Accepted scope`, `Assumptions`, `Out of scope`, and `Validation approach`.
For explicit fallback spec text, paste the full spec verbatim into the reviewer
prompt.
If no reliable spec can be supplied, stop and ask.

## PR Resolution

Resolve the PR non-interactively before review.

If a PR exists, use `gh pr view --json number,url,headRefName` and stop if its
head branch is not the checked-out branch.

If no PR exists:
- Push and freshness-check the branch first.
- Create the PR with `gh pr create --head <current-branch> --title ... --body-file ...`
  or an equivalent fully explicit non-interactive command.
- Do not use prompts, editors, `--fill`, or implicit push/fork behavior.
- Prefer `--body-file` over inline shell quoting for multi-section bodies. Do
  not let `gh` decide how to push or fork for you.
- Follow repo-local PR title/body rules. If title rules depend on commit
  prefixes, derive the title from consistent branch commit subjects; stop if
  they are inconsistent.
- If repo-local rules do not define a body shape, use the fallback in
  `/home/ar/repos/cc-config/skills/personal_dev/rocket/references/rocket-review-details.md`.
- Populate the PR body from the contract, landed changes, and validation that
  actually ran.

After creation, resolve the PR number/URL and verify the PR head branch.

## Completion Shortcut

After the PR exists, inspect existing comments before running reviewers. If any
comment contains this exact configured summary line, stop and report:

```text
review already complete
```

For `summary_title: Rocket Review Summary`, the line is:

```text
<summary>Rocket Review Summary</summary>
```

Do not add diary resume logic. One rocket review per PR is intentional; if Ar
wants a fresh review, Ar deletes the summary comment first.

## Practical Sequence

1. Resolve the selected review profile from config.
2. Preflight repo, branch, `gh`, configured runners, and repo-local rules.
3. Ensure the review target is pushed and upstream matches local `HEAD`.
4. Resolve or create the PR non-interactively.
5. Check the completion shortcut and stop if review is already complete.
6. Read `/home/ar/repos/cc-config/skills/personal_dev/rocket/references/rocket-review-details.md`.
7. Run configured reviewers in order.
8. After each round, decide patch/skip/open, commit and push fixes if needed,
   re-verify upstream freshness, then update the diary.
9. Determine the overall Rocket verdict from the final patched branch.
10. Post one final PR comment derived from the diary.
11. If a Linear ticket exists, sync the managed region.

## Review Rounds

Before constructing reviewer prompts, parsing output, writing the diary, posting
the final comment, or syncing Linear, you must read:

`/home/ar/repos/cc-config/skills/personal_dev/rocket/references/rocket-review-details.md`

Reviewer prompts must include the spec/contract, branch, PR number and URL,
repo/worktree path, configured slash command, and instructions to:
- review the branch against `Goal`, `Accepted scope`, `Assumptions`, and
  `Validation approach`
- respect `Out of scope`
- make round 1 an exhaustive discovery pass over the entire review target; do
  not stop after finding one actionable issue, and return every finding the
  reviewer can substantiate
- flag unnecessary complexity, non-idiomatic code, duplicate abstractions,
  brittle shortcuts, and simpler repo-native patterns that should have been used
- report non-blocking edge cases and hardening opportunities without withholding
  approval for them

A blocker must show that the current branch cannot safely deliver the core
ticket: broken goal or acceptance behavior, a concrete realistic in-scope edge
case with plainly incorrect behavior, or a credible security, data-loss,
data-corruption, or required-path concurrency failure. Distant or speculative
edge cases, defense-in-depth, maintainability, simplification, performance
outside expected scale, and out-of-scope improvements are non-blocking. They may
still be reported and patched.

Require reviewer output sections:
- `Critical`
- `High`
- `Low`
- `Uncertain`
- `Verdict`

The `Verdict` section must end with exactly one token: `APPROVE`,
`APPROVE WITH FIXES`, or `NEEDS FIXES`.

Run configured reviewers in strict order. Never run more than two rounds for one
reviewer, even if configuration requests more. Round 1 is the one full,
exhaustive review of the current pushed branch. If round 1 findings are patched,
pushed, and the reviewer has a second round remaining, round 2 is a focused
follow-up: give the reviewer its complete round 1 output plus the patch decisions
and commit, then ask whether the fixes are satisfactory. Round 2 must verify the
round 1 findings and flag unresolved findings or regressions caused by the
patches; it is not another from-scratch full review. If round 1 produced no
patch, do not rerun the reviewer against unchanged `HEAD`.

Never tell a round 2 reviewer to "inspect the entire branch independently" or
otherwise repeat round 1 discovery; use only the focused follow-up scope.

For each finding, choose exactly one diary status:
- `[patched]`
- `[skipped: not actionable]`
- `[skipped: reason]`
- `[open: blocker]`
- `[open: non-blocking]`

Patch findings that are validated against a credible code path and improve the
branch; do not patch mere possibilities only to satisfy a reviewer. Decide the
whole round first, then batch all accepted fixes into the single follow-up commit
for that round. Preserve reviewer severity buckets and exact verdict tokens.
Normalize only the common priority labels described in the details reference;
do not re-rank findings or rewrite a reviewer's verdict.

After the final round's accepted fixes are patched and verified, determine the
overall Rocket verdict from the final branch rather than the last raw reviewer
token. Approve when the core ticket is delivered and no validated unresolved
blocker remains. Non-blocking findings and findings patched after the second
round remain visible in the diary but do not prevent approval. Withhold approval
only for an `[open: blocker]`. This overall gate does not change the reviewer
verdict recorded for each round.

## Runner Failures

Allow the configured timeout for each round, default `900000` ms. Quiet periods
and progress chatter are normal while the process is still running.

Each failed round gets one automatic retry against the same pushed branch state.
If the retry fails, stop immediately and report the raw output, failure mode, and
actual elapsed time for both attempts. Do not synthesize a successful diary entry.

Use timeout language only when the full configured timeout was actually consumed.
Stopped-early runs are premature aborts, not timeouts.

## Artifacts

Maintain one diary file:

```text
_scratch/_reviews/<diary_name>_<branch-with-slashes-replaced-by-dashes>.md
```

The diary is the source of truth for the final PR comment. Keep it organized by
reviewer and round, preserve severity headings, include exact verdict tokens,
include the round commit hash for patched items, and record the overall Rocket
verdict plus any unresolved blockers.

At the end, post exactly one `gh pr comment` wrapped in a collapsed `<details>`
block using the configured `summary_title`. No claim in the PR comment may be
absent from the diary.

If a Linear ticket exists, update its description after the PR comment is posted.
Use the marker-bounded managed region shared with `rocket-plan`; do not post a
separate Linear comment.

## Stop Conditions

Stop and report instead of guessing if:
- repo, branch, profile, `gh`, auth, runner, push, upstream, or PR resolution fails
- the existing PR head branch differs from the checked-out branch
- repo-local PR title/body rules cannot be satisfied
- the working tree has unclear changes
- no reliable spec can be handed to reviewers
- a runner fails twice for the same round, times out, aborts early, or returns
  malformed output after normalization
