---
name: rocket-review
description: Run the final configured review loop for a completed branch, whether or not a PR already exists. Use this whenever the user says `rocket-review`, asks for the final review loop, or wants Codex to ensure the current branch has a PR, run the configured reviewers, patch what should be patched, and post one final PR summary comment. Optional usage: `rocket-review <profile>`.
---

# Rocket Review

Use this only after implementation is complete enough for external review.

This skill is narrow on purpose:
- It does not define the implementation work.
- It does not assign or reinterpret severity.
- It does not rely on interactive PR creation.
- It does not hardcode Claude, Codex, or Cursor behavior in the workflow body. Reviewer selection comes from the rocket config.

Your job is to take the current checked-out branch, ensure it has a PR, run the configured reviewer profile against the supplied spec, patch what should be patched, leave a strict audit trail, and post one final PR summary comment.

## Rocket Config

Before choosing reviewers, read the rocket config:

1. `skills/personal_dev/rocket/rocket.local.yaml` if it exists.
2. `skills/personal_dev/rocket/rocket.example.yaml` for defaults and for any profile missing from local config.

If the user invokes `rocket-review <profile>`, use `<profile>` as the review profile name.
If the user invokes bare `rocket-review`, use `defaults.review_profile`.

Stop if the selected review profile does not exist. Do not infer a profile from a hyphenated tool list.

Each review profile must provide:
- `slash_command`, usually `/code-review` or `/code-review-parallel`
- `summary_title`, used in the final PR comment disclosure summary
- `diary_name`, used for the local diary filename
- `reviewers`, an ordered list

Each reviewer must provide:
- `name`
- `runner`: one of `claude`, `codex`, or `cursor`
- optional `model`
- `max_rounds`

Runner invocation rules for review rounds:
- `claude`: `claude --dangerously-skip-permissions -p "$PROMPT"`
- `codex`: `codex exec --dangerously-bypass-approvals-and-sandbox "$PROMPT"`
- `cursor`: `cursor-agent -p -f "$PROMPT"`

When `model` is set, pass it with the runner's supported `--model <model>` flag.
For Cursor review rounds, keep `-p -f`; this matches the existing user convention for headless review.

Approval tokens:
- `APPROVE`
- `APPROVE WITH FIXES`
- `NEEDS FIXES`

A `## Verdict` ending in `APPROVE` or `APPROVE WITH FIXES` ends that reviewer phase early. `NEEDS FIXES` does not.

## Preconditions

Run these checks before PR resolution and round 1:

```bash
git rev-parse --is-inside-work-tree
git branch --show-current
command -v gh
gh auth status
git status -sb
```

Also check every configured runner:
- `claude` -> `command -v claude`
- `codex` -> `command -v codex`
- `cursor` -> `command -v cursor-agent`

Required conditions:
- You are inside the repo/worktree that contains the branch being reviewed.
- The intended review branch is the branch currently checked out.
- `gh` is available and authenticated.
- Every configured runner is available on `PATH` and authenticated enough to run non-interactively.

Before generating a PR title or PR body, read local repo rules first:
- `CLAUDE.md`
- `AGENTS.md`
- other nearby agent or workflow rules such as `.cursorrules`

Stop and report the problem if any precondition fails.

## Branch State

Each reviewer must review the actual pushed branch state, not a local-only draft.

Before round 1:
- If there are review-ready local changes that belong on this branch, commit them using the repo's normal commit conventions and push them before invoking any reviewer.
- If the working tree contains unrelated, ambiguous, or not-yet-ready changes, stop and ask the user instead of guessing.
- If the current branch has no upstream branch yet, push it before attempting PR creation.

After every push:
- verify that the upstream branch exists
- verify that local `HEAD` matches the upstream commit before creating a PR or invoking any reviewer
- stop if upstream is stale or missing

Between review rounds:
- If you patched anything in a round, make one follow-up commit for that round and push it before the next round.
- Each subsequent round must review the new pushed `HEAD`.
- Do not amend unless the user explicitly asks.
- Do not create extra bookkeeping commits.
- Do not rerun the same reviewer against an unchanged `HEAD`; if a reviewer returns `NEEDS FIXES` and you make no patch, record the remaining findings and move on after that round.

## Spec Contract

You must supply the spec to each reviewer in the prompt you construct.

Preferred spec source, in priority order:
- an implementation contract from `rocket-plan` persisted at `_scratch/_contracts/<branch>.md`
- a Linear ticket ID
- a full Linear ticket URL

Contract path rules:
- Use the raw branch path, not a flattened filename.
- Example: branch `aryan-binazir/BBA-11` maps to `_scratch/_contracts/aryan-binazir/BBA-11.md`.
- Treat `_scratch/_contracts/<branch>.md` as local review state by default. Do not require it to be committed, and do not commit `_scratch` artifacts unless the user explicitly asks.

The `rocket-plan` contract is the best review target because it contains:
- `Goal`
- `Accepted scope`
- `Assumptions`
- `Out of scope`
- `Validation approach`

Fallback:
- paste the full spec text verbatim into the prompt

When the local contract file exists, pass the reviewer its absolute file path in the prompt so it can open the contract directly if useful. You may also inline the contract contents; the key requirement is that the reviewer receives the contract explicitly rather than having to discover it.

Do not make reviewers discover the spec on their own. If you cannot supply a reliable spec, stop and ask the user.

## PR Resolution

The PR may or may not already exist. Resolve that non-interactively before review.

### If a PR already exists

- Use `gh pr view --json number,url,headRefName` for the current branch.
- Stop if the existing PR head branch does not match the checked-out branch.

### If no PR exists

Create one non-interactively. Do not rely on prompts, editors, or `--fill`.

Rules:
- Push and freshness-check the branch first.
- Use `gh pr create --head <current-branch> --title ... --body-file ...` or an equivalent fully explicit non-interactive form.
- Prefer `--body-file` over inline shell quoting for multi-section bodies.
- Do not let `gh` decide how to push or fork for you.

PR title:
- Follow repo-local rules if they exist.
- If repo-local rules tie PR titles to branch commit conventions, inspect the branch's commit subjects and derive the PR title from the consistent prefix.
- If the branch commits do not support a single consistent required prefix, stop instead of inventing one.

PR body:
- Follow repo-local required structure if it exists.
- Otherwise use this fallback:

```md
### Problem

### Changes

### Decisions

### Testing
How it was tested, or how to test it.
```

- Populate the body from the implementation contract, the code changes that actually landed, and the validation that actually ran.

After creating the PR:
- resolve the PR number and URL
- verify the PR head branch matches the checked-out branch

## Completion Shortcut

Once the PR exists, inspect its existing comments before review rounds begin.

If a comment already contains the exact configured summary line, stop and report:

```text
review already complete
```

For a profile with `summary_title: Rocket Review Summary`, the exact summary line is:

```text
<summary>Rocket Review Summary</summary>
```

Do not add diary resume logic. Treat this as the only completion shortcut.

## Review Prompt Contract

Construct each reviewer prompt yourself. The prompt must include:
- the implementation contract or fallback spec
- the current branch name
- the PR number and PR URL
- the repo/worktree path to review
- an explicit instruction to run the configured slash command
- an explicit request to review the current branch against `Goal`, `Accepted scope`, `Assumptions`, and `Validation approach`
- an explicit instruction to respect `Out of scope` items and not treat them as missing work
- an explicit request to flag unnecessary complexity, non-idiomatic code, duplicate abstractions, brittle shortcuts, and simpler existing repo patterns that should have been used

Require this exact output shape:
- `Critical`
- `High`
- `Low`
- `Uncertain`
- `Verdict`

Require the `Verdict` section to end with one of:
- `APPROVE`
- `APPROVE WITH FIXES`
- `NEEDS FIXES`

Do not ask for compliments, extra summary sections, or style feedback outside that structure.

## Prompt Template

Use a prompt equivalent to this for each reviewer:

```text
You are <reviewer.name> reviewing work completed on this branch.

Run the `<slash_command>` slash command for this review.

Review target:
- Repo/worktree: <absolute path>
- Branch: <branch>
- PR: #<number> <url>

Implementation contract:
<contents of _scratch/_contracts/<branch>.md, or fallback spec text>

Review against:
- Goal
- Accepted scope
- Assumptions
- Validation approach

Respect Out of scope items. Do not treat them as missing work.

Also review implementation quality. Flag any case where the branch solved the problem in a sloppy, overcomplicated, non-idiomatic, or brittle way. Call out simpler existing repo patterns, helpers, abstractions, or integration points that should have been used instead.

Review only the changes introduced on this branch. The configured slash command handles scoping.

Give a brutally honest review of whether the current branch satisfies the contract and whether it used the simplest repo-idiomatic implementation path.

Return findings grouped exactly as:
## Critical
## High
## Low
## Uncertain
## Verdict

The Verdict section must end with one of these exact tokens on its own line:
- APPROVE
- APPROVE WITH FIXES
- NEEDS FIXES

Use APPROVE when the branch is ready to merge as-is.
Use APPROVE WITH FIXES when the branch is acceptable but you are requesting specific fixes that the implementer should apply before merge.
Use NEEDS FIXES when the branch is not yet acceptable.

Within each finding, include concrete file and line references when possible.
No padding. No compliments.
```

## Runner Output Handling

If a reviewer returns priority-style findings instead of the requested section headings, normalize them rather than failing immediately. Treat these labels as equivalent severities:
- `P0` -> `Critical`
- `P1` -> `High`
- `P2` or `P3` -> `Low`
- findings without a usable priority, or hedged/design-observation findings without a clear severity -> `Uncertain`

If a reviewer returns a freeform review plus one or more `P0`/`P1`/`P2`/`P3` findings, extract those findings, map them into the standard severity buckets above, and continue the review loop.

Do not invent verdict tokens. If the `## Verdict` section is missing or empty after normalization, treat the output as malformed.

## Reviewer Execution Contract

Each reviewer round, regardless of runner, must follow these rules.

Timeout:
- Allow up to the full 15-minute budget per round by default: `900000` ms, unless the profile or reviewer config explicitly sets another timeout.
- Record the launch timestamp when the CLI starts.
- If your tooling supports one blocking wait for the full budget, prefer that.
- If polling, recalculate `remaining_budget_ms = timeout_ms - elapsed_ms` after each poll and keep waiting until either the process exits or the budget is exhausted.
- Never use a short fixed poll schedule whose total explicit waits add up to less than the full timeout.
- Quiet periods are normal. Progress chatter, plugin warnings, `collab:` events, retry noise, or other intermediate logs are not by themselves timeout evidence and are not malformed-output evidence while the process is still running.
- If the workflow stops waiting before the full timeout elapses and before the CLI reaches a terminal result, classify that as a premature abort. Do not describe it as a timeout.

Failure modes:
- `premature abort`: the workflow stopped waiting before the full budget elapsed and before a terminal result.
- `timeout`: the CLI was still running after the full timeout elapsed.
- `process failure`: the CLI exited non-zero.
- `malformed output`: the CLI exited within budget, but the final output contains neither the expected `Critical`, `High`, `Low`, `Uncertain`, and `Verdict` sections nor any parsable `P0`/`P1`/`P2`/`P3` findings that can be normalized, or the `Verdict` section is missing or empty after normalization.

Retry:
- A failed round is eligible for exactly one automatic retry within the same round number, using the same prompt against the same pushed branch state.
- If the retry also fails, stop the review loop immediately, report the raw output, exact failure mode, and actual elapsed time for both attempts, and do not synthesize a diary entry pretending the round succeeded.
- Each round number gets one original attempt and one retry. Do not consume the next round's budget as an additional retry.

Output handling:
- Capture the complete CLI output for the whole run, including progress chatter and the eventual final answer.
- If progress logs appear before the final answer, ignore that leading noise and extract the final structured review block from the completed output.
- If the CLI exits successfully with a freeform review plus parseable `P0`/`P1`/`P2`/`P3` findings, normalize those findings instead of failing on formatting.
- Do not call a run malformed just because early or intermediate output lacks the required headings.

Failure wording:
- Only use timeout language if the run really consumed the full configured timeout budget.
- If the run stopped earlier than that, say it was stopped early or prematurely aborted, and include the actual elapsed time.
- If the process exited on its own before the budget without valid final sections, call it malformed output or process failure as appropriate, not a timeout.

## Verdict Parsing

The agent decides whether to keep looping based strictly on the parsed verdict.

Rules:
- Locate the `## Verdict` section in the completed reviewer output.
- Extract the verdict token as the last non-empty line under that section, normalized to uppercase with surrounding whitespace and trailing punctuation stripped.
- Approval is reached when the normalized token equals `APPROVE` or `APPROVE WITH FIXES`.
- Any other token (including `NEEDS FIXES`, `REJECT`, or anything unrecognized) is treated as non-approval.
- If the `## Verdict` section is missing or empty, treat it as malformed output and follow the retry rule above. Do not infer a verdict from severity counts.
- Do not invent verdict tokens. Do not collapse `APPROVE WITH FIXES` into `APPROVE` in the diary or PR comment; preserve the exact token the reviewer returned.
- Severity counts (empty Critical/High etc.) do not by themselves end the loop. The verdict token does.

## Review Loop

The loop runs configured reviewers in strict order. Each reviewer has `max_rounds` and exits early on any approving verdict.

For each reviewer:
1. Run round 1 against the current pushed branch state.
2. Read findings conservatively. Err toward patching rather than dismissing.
3. Patch what should be fixed.
4. For each finding, decide one of:
   - `[patched]`
   - `[skipped: not actionable]`
   - `[skipped: reason]`
   - `[open]`
5. If you patched anything, create one follow-up commit for that round and push it.
6. Re-verify that upstream matches local `HEAD`.
7. Update the diary for that reviewer round.
8. If the reviewer returned `APPROVE` or `APPROVE WITH FIXES`, end that reviewer phase and move to the next configured reviewer.
9. If the reviewer returned `NEEDS FIXES`, patched changes were pushed, and the reviewer has remaining `max_rounds`, run the next round for that same reviewer.
10. Stop that reviewer phase after `max_rounds`, or earlier if no patch was made against a `NEEDS FIXES` result.

After all reviewer phases:
- Any unresolved finding that still matters and is not intentionally dismissed with a reason must be marked `[open]`.
- Do not run reviewers that are not listed in the selected profile.
- Do not run extra rounds beyond the configured `max_rounds`.

## Severity Ownership

Severity comes from the configured slash command, not from you.

Your responsibilities are:
- preserve each reviewer's severity buckets exactly when it uses `Critical`, `High`, `Low`, and `Uncertain`
- when a reviewer emits `P0`/`P1`/`P2`/`P3`, normalize them using the mapping defined above without reinterpretation
- preserve the exact verdict token per round
- decide what to patch
- decide what to skip with a reason
- mark anything still unresolved after the final allowed round as `[open]`

Do not:
- rename severity levels
- collapse severity levels into custom buckets
- re-rank findings just because you disagree with the emphasis
- promote a `NEEDS FIXES` verdict to `APPROVE` because you patched everything; only the reviewer's own next-round verdict can approve its phase

## Diary

Maintain one diary file as the source of truth:

```text
_scratch/_reviews/<diary_name>_<branch-safe>.md
```

Use the branch name as the identity. For the filename only, replace `/` with `-` so the file stays flat.

Create `_scratch/_reviews` if needed.

Use reviewer-and-round-level sections, not per-finding lifecycle logs.

Required structure:

```md
# Rocket Review: <branch>

## Cursor Round 1
### Verdict: NEEDS FIXES

### Critical
- [file:line] - description [patched] (commit abc123)

### High
- [file:line] - description [skipped: reason]

### Low
- [file:line] - description [open]

### Uncertain
- (none)
```

Rules:
- Preserve severity grouping exactly as the reviewer returned it.
- Keep each round self-contained.
- If a severity group has no items, write `- (none)`.
- Include the round commit hash when an item was patched in that round.
- If a later round surfaces a new finding caused by an earlier patch, note that explicitly in the finding text instead of inventing a new status.
- Record the exact verdict token in the round's `### Verdict:` line.
- Do not claim a patch, skip, or open item unless it happened in that round.

## Final PR Comment

Post exactly one PR comment at the end, derived strictly from the diary.

Use `gh pr comment` against the current branch's PR.

Required shape:

```md
<details>
<summary><summary_title></summary>

**Profile:** <profile>
**Rounds:** <reviewer round count>
**Cursor verdict:** APPROVE WITH FIXES
**Codex verdict:** APPROVE

### Cursor
#### Critical
- [file:line] - description [patched]

#### High
- [file:line] - description [skipped: reason]

#### Low
- [file:line] - description [open]

### Codex
#### Critical
- (none)

</details>
```

Rules:
- Wrap the whole PR comment body in a closed GitHub disclosure block using `<details>` and `<summary><summary_title></summary>`.
- Do not add the `open` attribute; the disclosure must render collapsed by default.
- Use each configured reviewer's name as the reviewer section.
- Each reviewer verdict line must be the verdict token from that reviewer's final round that ran. If a reviewer ended on `NEEDS FIXES` after its final allowed round, report `NOT APPROVED (NEEDS FIXES after <n> rounds)` for that reviewer.
- Do not write `APPROVE` in a verdict unless the diary records an approving verdict from that reviewer.
- No claim in the PR comment may be absent from the diary.
- Preserve severity headings.
- Use `[patched]`, `[skipped: reason]`, and `[open]` exactly.
- No padding. No compliments.

## Linear Ticket Sync

Skip this step if no Linear ticket exists.

After all review rounds are done and the final PR comment is posted, update the Linear ticket description. Do not post this as a separate ticket comment.

Use the same marker-bounded managed region as `rocket-plan`:
- look for `<!-- managed:rocket-start -->` and `<!-- managed:rocket-end -->` in the description
- if both markers exist, replace everything between them, inclusive of markers
- if markers are missing, append the managed region to the end of the description
- never touch content outside the markers
- if only one marker is found, treat it as missing and append a fresh managed region

When rebuilding the managed region:
- always emit both `<!-- managed:rocket-start -->` and `<!-- managed:rocket-end -->` markers
- if the implementation contract exists, include the current `## Rocket Plan Contract` block first
- if no implementation contract exists, preserve the existing `## Rocket Plan Contract` block from the current description inside the markers
- then include exactly one `Rocket Review` section
- do not create duplicate managed regions or duplicate review sections

For the review section:
- first verify the exact currently supported Linear collapsible-section syntax against official Linear editor documentation in the current session
- do not assume `>>>` or `<details>` from memory
- if collapsible syntax is clearly verified, use a collapsed section titled `Rocket Review`
- if verification is unclear, fall back to a plain `## Rocket Review` section instead of emitting broken markdown

Content requirements:
- include what each reviewer found, what was patched, what was skipped, and why skipped items were left as-is
- include each reviewer's final verdict
- keep the ticket description as the source of truth for the final reviewed state

## Practical Sequence

Use this order:
1. Resolve the selected review profile from config.
2. Verify repo, branch, `gh`, configured runners, and local repo rules.
3. Make sure the review target is the current pushed branch state.
4. Resolve the PR for the current branch, creating it non-interactively if needed.
5. Check PR comments for the existing configured summary line; if found, stop and report `review already complete`.
6. Build the shared prompt template with the implementation contract or fallback spec, branch, PR, repo path, and configured slash command.
7. Run configured reviewers in order, obeying each reviewer's runner, model, and `max_rounds`.
8. After each round, update the diary after patch/skip/open decisions are made.
9. If needed, commit and push fixes, then re-verify upstream freshness before the next round.
10. Derive one final PR comment from the diary and post it.
11. If a Linear ticket exists, update the ticket description with the managed contract/review tail.

## Stop Conditions

Stop immediately and report back instead of guessing if:
- you are not in a git repo/worktree
- the current branch cannot be resolved
- the selected review profile cannot be resolved
- `gh` is unavailable or unauthenticated
- a configured runner is unavailable or not authenticated
- the current branch cannot be pushed or upstream cannot be made fresh
- no PR exists and deterministic `gh pr create --head ... --title ... --body-file ...` also fails
- an existing PR's head branch does not match the checked-out branch
- repo-local PR title rules cannot be satisfied from the branch commit history
- a runner CLI call exits non-zero twice for the same round, times out, is prematurely aborted, or returns malformed output
- the spec was not provided in a form you can hand to the reviewer
- the working tree contains unclear changes you cannot safely include in the review

## What This Skill Does Not Do

- It does not skip the review phase.
- It does not merge the PR.
- It does not replace repo-local rules.
- It does not keep the contract only in session memory.
- It does not ask reviewers to discover the spec.
- It does not silently fall back to a different profile.
- It does not run CodeRabbit. That workflow is retired.
