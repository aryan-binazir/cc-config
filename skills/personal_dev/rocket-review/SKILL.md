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
strict local diary, and post exactly one final PR summary comment. Scope stops
there: implementation work, reworded severities or verdict tokens, merging, and
profile switches stay outside this skill.

## Config

Resolve this `SKILL.md` to its real path first, then resolve `../rocket`
relative to its directory and call that absolute path `<rocket-dir>`. Use it
for every Rocket script and reference below so symlinked installs, home
directories, and operating systems leave the workflow unchanged.

Run `uv run --script <rocket-dir>/scripts/resolve_config.py` before choosing
reviewers. It reads `rocket.local.yaml` over `rocket.example.yaml`; once it
succeeds, its output is the only config source.

Use `rocket-review <profile>` when provided; otherwise
`defaults.review_profile`. The profile comes only from that literal argument or
default — stop if the selected `review_profiles.<profile>` is missing.

Each review profile provides `slash_command`, `summary_title`, `diary_name`,
and ordered `reviewers`. Each reviewer provides `name`, `runner`, optional
`model`, optional Claude `effort`, optional Codex `reasoning_effort`, optional
`timeout_ms`, and `max_rounds`.

Runner commands:
- For Cursor, resolve the `call-cursor` skill's `SKILL.md` to its real path;
  `<call-cursor-skill-dir>` is that file's directory.
- `claude`: `claude --permission-mode auto -p "$PROMPT"`
- `codex`: `codex --sandbox read-only --ask-for-approval on-request -c approvals_reviewer=auto_review exec "$PROMPT" < /dev/null`
- `cursor`: `uv run --script "<call-cursor-skill-dir>/scripts/call.py" "$PROMPT"`

When `model` is set, pass the runner's supported `--model <model>` flag; for
Cursor pass it to the wrapper. Pass Cursor `timeout_ms` to the wrapper as
`--timeout-ms <timeout_ms>`. Pass Claude `effort` as `--effort <effort>` and
Codex `reasoning_effort` as
`-c model_reasoning_effort="<reasoning_effort>"`; stop if either option is
configured for a mismatched runner.

Reviewers are read-only, invoked through exactly the commands above. The Cursor
wrapper fails closed unless CLI Auto-review is active; stop if the installed CLI
lacks it. The reviewer prompt must
state that the review is read-only and files stay unmodified — patching
findings is the main agent's job.

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

Read repo-local rules before generating a PR title or body: `CLAUDE.md`,
`AGENTS.md`, and nearby workflow rules such as `.cursorrules`.

Stop if the repo/worktree, branch, `gh`, auth, runner availability, or runner
non-interactive auth is short of ready.

If repo rules require `_scratch/_context/<branch>.md`, update it when review
plans, assumptions, decisions, fixes, or final review state change.

## Branch State

Reviewers review the actual pushed branch state.

Before round 1:
- Commit review-ready local changes that belong on this branch using repo
  conventions, and push before invoking reviewers.
- Stop and ask about local changes that are unrelated, ambiguous, or short of
  review-ready.
- If there is no upstream branch yet, push before PR creation.

After every push, verify the upstream branch exists and local `HEAD` matches
it; stop if upstream is stale or missing.

After a review round:
- Patched findings go in one follow-up commit for that round, pushed. Amend
  only when the user asks, and commit only real fixes.
- Rerun a reviewer only against a newly pushed `HEAD`; against unchanged
  `HEAD`, record unresolved findings and move on after that round.
- An approval verdict ends that reviewer phase even if an accepted patch
  changes `HEAD`; the approved review stays closed.

## Spec Source

Supply the spec directly to every reviewer, discovery-free. Use the best
available source in this order:

1. `_scratch/_contracts/<branch>.md` from `rocket-plan`
2. a Linear or Jira ticket ID, resolving the tracker with available tooling
   (the key format fits both trackers)
3. a full Linear or Jira ticket URL
4. a markdown spec file path supplied by Ar
5. explicit fallback spec text

Contracts use the raw branch path: `aryan-binazir/BBA-11` maps to
`_scratch/_contracts/aryan-binazir/BBA-11.md`. `_scratch` review artifacts are
local state; commit them only when the user explicitly asks.

When a local contract exists, pass its absolute path to the reviewer and
include or summarize its contents in the prompt; it must include the review
target: `Goal`, `Accepted scope`, `Assumptions`, `Out of scope`, and
`Validation approach`. Paste explicit fallback spec text verbatim into the
reviewer prompt. If no reliable spec can be supplied, stop and ask.

## PR Resolution

Resolve the PR non-interactively before review.

If a PR exists, use `gh pr view --json number,url,headRefName` and stop if its
head branch differs from the checked-out branch.

If no PR exists:
- Push and freshness-check the branch first.
- Create the draft PR with a fully explicit non-interactive command:
  `gh pr create --draft --head <current-branch> --title ... --body-file ...`.
  Spell out every push/fork decision in flags yourself, and prefer
  `--body-file` over inline shell quoting for multi-section bodies.
- Follow repo-local PR title/body rules. If title rules depend on commit
  prefixes, derive the title from consistent branch commit subjects and stop if
  they are inconsistent.
- If repo-local rules leave the body shape open, use the fallback in
  `<rocket-dir>/references/rocket-review-details.md`.
- Populate the PR body from the contract, landed changes, and validation that
  actually ran.

After creation, resolve the PR number/URL and verify the PR head branch.

## Completion Shortcut

After the PR exists, inspect existing comments before running reviewers. If any
comment contains this exact configured summary line, stop and report
`review already complete`. For `summary_title: Rocket Review Summary`, the line
is:

```text
<summary>Rocket Review Summary</summary>
```

One rocket review per PR is intentional; a fresh review starts with Ar deleting
the summary comment.

## Practical Sequence

1. Resolve the selected review profile from config.
2. Preflight repo, branch, `gh`, configured runners, and repo-local rules.
3. Ensure the review target is pushed and upstream matches local `HEAD`.
4. Resolve or create the PR non-interactively.
5. Check the completion shortcut.
6. Read `<rocket-dir>/references/rocket-review-details.md`.
7. Run configured reviewers in order.
8. After each round, decide patch/skip/open, commit and push fixes if needed,
   re-verify upstream freshness, then update the diary.
9. Record every executed round's exact reviewer verdict and any post-round
   branch state.
10. Post one final PR comment derived from the diary.
11. If a Linear ticket exists, sync the managed region.
12. Return the final user-facing status grouped by numbered PR, with every
    reviewer named beside that reviewer's exact round result.

## Review Rounds

Read `<rocket-dir>/references/rocket-review-details.md` before constructing
reviewer prompts, parsing output, writing the diary, posting the final comment,
or syncing Linear.

Reviewer prompts include the spec/contract, branch, PR number and URL,
repo/worktree path, configured slash command, and instructions to:
- review the branch against `Goal`, `Accepted scope`, `Assumptions`, and
  `Validation approach`, respecting `Out of scope`
- make round 1 an exhaustive discovery pass over the entire review target,
  returning every finding the reviewer can substantiate
- flag unnecessary complexity, non-idiomatic code, duplicate abstractions,
  brittle shortcuts, and simpler repo-native patterns that should have been used
- report non-blocking edge cases and hardening opportunities while still
  granting approval

A blocker must show that the current branch cannot safely deliver the core
ticket: broken goal or acceptance behavior, a concrete realistic in-scope edge
case with plainly incorrect behavior, or a credible security, data-loss,
data-corruption, or required-path concurrency failure. Distant or speculative
edge cases, defense-in-depth, maintainability, simplification, performance
outside expected scale, and out-of-scope improvements are non-blocking; they
may still be reported and patched.

Required reviewer output sections: `Critical`, `High`, `Low`, `Uncertain`,
`Verdict`. The `Verdict` section ends with exactly one token: `APPROVE`,
`APPROVE WITH FIXES`, or `NEEDS FIXES`.

Run configured reviewers in strict order, two rounds per reviewer at most —
even when configuration requests more. Round 1 is the one full, exhaustive
review of the current pushed branch. `APPROVE` and `APPROVE WITH FIXES` are
approval verdicts and end that reviewer's rounds: apply any accepted fixes,
commit, push, and record them as post-round branch state, leaving the approving
reviewer finished.

Only a non-approval round 1 verdict can trigger round 2, and only when accepted
fixes are patched and pushed and the reviewer has a second round remaining
(`max_rounds` above `1`); otherwise the reviewer phase ends after round 1.
Round 2 is one focused follow-up: give the reviewer its complete round 1 output
plus the patch decisions and commit, and ask whether the fixes are
satisfactory — it verifies the round 1 findings and flags unresolved findings
or regressions caused by the patches. Keep round 2 scoped to exactly that
follow-up; full-branch discovery stays a round 1 activity. When the configured
slash command is `/code-review`, identify the round 2 command as
`/code-review single` so its default parallel discovery workflow stays off — a
round-scoped prompt change that leaves the saved reviewer configuration as-is.

For each finding, choose exactly one diary status: `[patched]`,
`[skipped: not actionable]`, `[skipped: reason]`, `[open: blocker]`,
`[open: non-blocking]`.

Patch only findings validated against a credible code path that improve the
branch; a mere possibility raised to satisfy a reviewer stays unpatched. Decide
the whole round first, then batch all accepted fixes into that round's single
follow-up commit. Preserve reviewer severity buckets, ranking, and exact
verdict tokens; normalize only the common priority labels described in the
details reference.

Every user-facing review report and artifact displays every executed round with
that round's exact reviewer verdict token — per-round, per-reviewer. Never
collapse rounds into a per-reviewer final verdict, and never derive or display
an overall Rocket verdict. Findings patched after an approval verdict or after
a reviewer's final allowed round are reported separately as post-round branch
state, explicitly marked as not re-reviewed; the reviewer's recorded verdict
stays as-is and covers only what it reviewed.

## Runner Failures

Allow the configured timeout for each round, default `900000` ms. Quiet periods
and progress chatter are normal while the process is still running.

Each failed round gets one automatic retry against the same pushed branch
state. If the retry fails, stop immediately and report the raw output, failure
mode, and actual elapsed time for both attempts — the diary records the failure
exactly as it happened.

Use timeout language only when the full configured timeout was actually
consumed; stopped-early runs are premature aborts.

## Artifacts

Maintain one diary file:

```text
_scratch/_reviews/<diary_name>_<branch-with-slashes-replaced-by-dashes>.md
```

A run starts fresh, overwriting any existing diary. The diary is the source of
truth for the final PR comment. Keep it organized by
reviewer and round; preserve severity headings; include exact verdict tokens
and the round commit hash for patched items; list every round in the review
ledger; and record unresolved blockers plus any post-round branch state.
Verdicts stay per-round only.

At the end, post exactly one `gh pr comment` wrapped in a collapsed `<details>`
block using the configured `summary_title`. Every claim in the PR comment
traces back to the diary.

If a Linear ticket exists, update its description after the PR comment is
posted, writing only inside the marker-bounded managed region shared with
`rocket-plan` — that managed-region edit is the sole Linear write.

## User-Facing Completion Report

The final assistant response is PR-first and reviewer-explicit. For every PR in
the task, use this exact grouping:

```text
PR 1 — <repo or service> #<number>: <url>
- Cursor — 1st round: `NEEDS FIXES`; 2nd round: `APPROVE WITH FIXES`
- Codex — 1st round: `NEEDS FIXES`; 2nd round: `NEEDS FIXES`
- Post-review patch — <what was patched after which named reviewer round>;
  not re-reviewed by <reviewer>
- Unresolved blockers — None

PR 2 — <repo or service> #<number>: <url>
- Cursor — 1st round: `NEEDS FIXES`; 2nd round: `APPROVE`
- Codex — 1st round: `NEEDS FIXES`; 2nd round: `APPROVE`
- Unresolved blockers — None
```

Rules:
- Number PR groups in task order — `PR 1` even when the task has only one PR —
  and keep every PR's status separate.
- Name the configured reviewer on every result line, showing every executed
  round and its exact verdict; a reviewer that ran once shows only the 1st
  round.
- Keep each post-round patch inside its PR group, naming the reviewer and round
  after which it landed and stating explicitly when it was not re-reviewed.
- Keep unresolved blockers inside their PR group.
- The structure is required; a short implementation or testing summary may
  follow the PR groups.

## Stop Conditions

Stop and report instead of guessing if:
- repo, branch, profile, `gh`, auth, runner, push, upstream, or PR resolution
  fails
- the existing PR head branch differs from the checked-out branch
- repo-local PR title/body rules cannot be satisfied
- the working tree has unclear changes
- no reliable spec can be handed to reviewers
- a runner fails twice for the same round, times out, aborts early, or returns
  malformed output after normalization
