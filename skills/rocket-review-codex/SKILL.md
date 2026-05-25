---
name: rocket-review-codex
description: Run the final Codex review loop for a completed branch, whether or not a PR already exists. Use this whenever the user explicitly says `rocket-review-codex`, asks for the Codex-only review loop, or wants the agent to ensure the current branch has a PR, run up to two harsh Codex review rounds against the supplied spec until Codex returns an `APPROVE` verdict (conditional approvals count), patch what should be patched, and post one final PR summary comment that surfaces whether the branch was approved.
---

# Rocket Review (Codex)

Use this only after implementation is complete enough for external review.

This skill is narrow on purpose:
- It does not define the implementation work.
- It does not assign or reinterpret severity.
- It does not run more than 2 external review rounds.
- It does not rely on interactive PR creation.
- Codex is the sole reviewer.

Your job is to take the current checked-out branch, ensure it has a PR, run up to 2 harsh detached Codex review rounds against the supplied spec until Codex returns an `APPROVE` verdict, patch what should be patched, leave a strict audit trail, and post one final PR summary comment whose final verdict reflects whether Codex approved.

A `## Verdict` containing `APPROVE` ends the loop, including conditional forms such as `APPROVE WITH FIXES`. Any other verdict (for example `NEEDS FIXES`) does not.

## Preconditions

Run these checks before PR resolution and round 1:

```bash
git rev-parse --is-inside-work-tree
git branch --show-current
command -v gh
gh auth status
git status -sb
command -v codex
```

Required conditions:
- You are inside the repo/worktree that contains the branch being reviewed.
- The intended review branch is the branch currently checked out.
- `gh` is available and authenticated.
- `codex` is available on `PATH`.

Before generating a PR title or PR body, read local repo rules first:
- `CLAUDE.md`
- `AGENTS.md`
- other nearby agent or workflow rules such as `.cursorrules`

Stop and report the problem if any precondition fails.

## Branch State

Each reviewer must review the actual pushed branch state, not a local-only draft.

Before round 1:
- If there are review-ready local changes that belong on this branch, commit them using the repo's normal commit conventions and push them before asking Codex to review.
- If the working tree contains unrelated, ambiguous, or not-yet-ready changes, stop and ask the user instead of guessing.
- If the current branch has no upstream branch yet, push it before attempting PR creation.

After every push:
- verify that the upstream branch exists
- verify that local `HEAD` matches the upstream commit before creating a PR or calling Codex
- stop if upstream is stale or missing

After round 1:
- If you patched anything, make one follow-up commit for that round and push it before round 2.
- Do not amend unless the user explicitly asks.
- Do not create extra bookkeeping commits.

## Spec Contract

You must supply the spec to Codex in the prompt you construct.

Preferred spec source (in priority order):
- an implementation contract from `rocket_plan` persisted at `_scratch/_contracts/<branch>.md`
- a Linear ticket ID
- a full Linear ticket URL

Contract path rules:
- Use the raw branch path, not a flattened filename.
- Example: branch `aryan-binazir/BBA-11` maps to `_scratch/_contracts/aryan-binazir/BBA-11.md`.
- Treat `_scratch/_contracts/<branch>.md` as local review state by default. Do not require it to be committed, and do not commit `_scratch` artifacts unless the user explicitly asks.

The `rocket_plan` contract is the best review target because it contains:
- `Goal`
- `Accepted scope`
- `Assumptions`
- `Out of scope`
- `Validation approach`

Fallback:
- paste the full spec text verbatim into the prompt

When the local contract file exists, pass Codex its absolute file path in the prompt so it can open the contract directly if useful. You may also inline the contract contents; the key requirement is that the reviewer receives the contract explicitly rather than having to discover it.

Do not make Codex discover the spec on its own. If you cannot supply a reliable spec, stop and ask the user.

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

If a comment already contains the exact summary line `<summary>Rocket Review Summary</summary>`, stop and report:

```text
review already complete
```

Do not add diary resume logic. Treat this as the only completion shortcut.

## Review Prompt Contract

Construct the Codex prompt yourself. The prompt must include:
- the implementation contract or fallback spec
- the current branch name
- the PR number and PR URL
- the repo/worktree path to review
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

## Codex Prompt Contract

Every review round uses detached Codex against the current pushed branch state.

Construct the `codex exec --dangerously-bypass-approvals-and-sandbox` prompt yourself. Add an explicit instruction to use `/code-review-parallel`.

## Codex Prompt Template

Use a prompt equivalent to this:

```text
You are Codex reviewing work completed on this branch.

Run the `/code-review-parallel` slash command for this review.

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

Review only the changes introduced on this branch. The `/code-review-parallel` command handles scoping.

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

`codex exec --dangerously-bypass-approvals-and-sandbox` may be easier to drive with a heredoc:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
codex exec --dangerously-bypass-approvals-and-sandbox "$PROMPT"
```

If detached Codex returns priority-style findings instead of the requested section headings, normalize them rather than failing immediately. Treat these labels as equivalent severities:
- `P0` -> `Critical`
- `P1` -> `High`
- `P2` or `P3` -> `Low`
- findings without a usable priority, or hedged/design-observation findings without a clear severity -> `Uncertain`

If Codex returns a freeform review plus one or more `P0`/`P1`/`P2`/`P3` findings, extract those findings, map them into the standard severity buckets above, and continue the review loop.

**important** Timeout rules:
- Allow up to the full 15-minute budget for each `codex exec --dangerously-bypass-approvals-and-sandbox` run: `900000` ms.
- Do not stop early just because Codex has been quiet for a few minutes.
- If a review run exceeds the full `900000` ms budget, treat it as a timeout failure.

## Detached Codex Execution Contract

Every Codex round's timeout handling must be explicit and budget-based, not vibe-based.

- Treat each detached Codex round as a single job with a total wall-clock budget of `900000` ms.
- Record the launch timestamp when `codex exec --dangerously-bypass-approvals-and-sandbox` starts. Every later wait, poll, or classification decision must measure elapsed time against that original launch timestamp.
- If your tooling supports one blocking wait for `900000` ms, prefer that.
- If your tooling requires polling, recalculate `remaining_budget_ms = 900000 - elapsed_ms` after each poll and keep waiting until either:
  - the process exits, or
  - `remaining_budget_ms <= 0`
- Never use a short fixed poll schedule whose total explicit waits add up to less than `900000` ms. A sequence like `30s + 60s + 60s` is a premature abort, not a timeout policy.
- Quiet periods are normal. Progress chatter, plugin warnings, `collab: SpawnAgent`, `collab: Wait`, retry noise, or other intermediate logs are not by themselves timeout evidence and are not malformed-output evidence while the process is still running.
- Do not classify the output as malformed while `codex exec --dangerously-bypass-approvals-and-sandbox` is still running and has remaining budget. Only validate the final output shape after the process exits, or after the full `900000` ms budget is actually exhausted.
- If the workflow, operator, or wrapper stops waiting before `900000` ms elapse and before `codex exec --dangerously-bypass-approvals-and-sandbox` reaches a terminal result, classify that as a premature abort. Do not describe it as "Codex timed out after 15 minutes."

## Verdict Parsing

The agent decides whether to keep looping based strictly on the parsed Codex verdict.

Rules:
- Locate the `## Verdict` section in the completed Codex output.
- Extract the verdict token as the last non-empty line under that section, normalized to uppercase with surrounding whitespace and trailing punctuation stripped.
- Approval is reached when the normalized token equals `APPROVE` or `APPROVE WITH FIXES`.
- Any other token (including `NEEDS FIXES`, `REJECT`, or anything unrecognized) is treated as non-approval.
- If the `## Verdict` section is missing or empty, treat it as malformed output and follow the failure handling rules below. Do not infer a verdict from severity counts.
- Do not invent verdict tokens. Do not collapse `APPROVE WITH FIXES` into `APPROVE` in the diary or PR comment; preserve the exact token Codex returned.
- Severity counts (empty Critical/High etc.) do not by themselves end the loop. The verdict token does.

## Review Loop

Maximum review rounds: 2.

For each round (round 1, then round 2 if not yet approved):

1. Run detached Codex review against the current pushed branch state.
2. Read the findings conservatively. Err toward patching rather than dismissing.
3. Patch what should be fixed.
4. For each finding, decide one of:
   - `[patched]`
   - `[skipped: not actionable]`
   - `[skipped: reason]`
5. If you patched anything, create one commit for this round and push it.
6. Re-verify that upstream matches local `HEAD`.
7. Record the round in the diary, including the exact Codex verdict token.
8. Inspect the verdict:
   - If the verdict is `APPROVE` or `APPROVE WITH FIXES`, exit the loop. Conditional approval still ends the loop after the round's patches are committed and pushed.
   - Otherwise, if this was round 1, continue to round 2.
   - Otherwise (round 2 ended without approval), stop the loop and proceed to the final PR comment with a NOT APPROVED final verdict.

Rules:
- Even if round 1 produces no code changes, round 2 still runs unless round 1 already returned `APPROVE` or `APPROVE WITH FIXES`.
- Do not run a third round under any circumstances.
- After the final allowed round, any unresolved finding that still matters and is not intentionally dismissed with a reason must be marked `[open]`.

## Codex CLI Failure Handling

Distinguish these failure modes precisely:
- `premature abort`: the workflow, operator, or wrapper stopped waiting before the full `900000` ms budget elapsed and before `codex exec --dangerously-bypass-approvals-and-sandbox` produced a terminal result
- `timeout`: `codex exec --dangerously-bypass-approvals-and-sandbox` was still running after the full `900000` ms budget elapsed
- `process failure`: `codex exec --dangerously-bypass-approvals-and-sandbox` exited non-zero
- `malformed output`: `codex exec --dangerously-bypass-approvals-and-sandbox` exited within budget, but the final collected output contains neither the expected `Critical`, `High`, `Low`, `Uncertain`, and `Verdict` sections nor any parsable `P0`/`P1`/`P2`/`P3` findings that can be normalized into those sections, or the `Verdict` section is missing or empty after normalization

Output handling rules:
- Capture the complete detached Codex output for the whole run, including progress chatter and the eventual final answer.
- If progress logs appear before the final answer, ignore that leading noise and extract the final structured review block from the completed output.
- If Codex exits successfully with a freeform review plus parseable `P0`/`P1`/`P2`/`P3` findings, normalize those findings instead of failing on formatting.
- Do not call a run malformed just because early/intermediate output lacks the required headings.

Retry rules:
- A failed Codex round (`premature abort`, `timeout`, `process failure`, or `malformed output`) is eligible for exactly one automatic retry within the same round number.
- The retry uses the same prompt against the same pushed branch state.
- If the retry also fails, stop the review loop immediately:
  - do not guess at missing structure
  - report the raw Codex output, exact failure mode, and actual elapsed time for both attempts to the user
  - do not write a synthesized diary entry pretending the review succeeded
- Do not consume the next round's budget as an additional retry. Each round number gets one original attempt and one retry, total.

Failure wording rules:
- Only use timeout language if the run really consumed the full `900000` ms budget.
- If the run stopped earlier than that, say it was stopped early or prematurely aborted, and include the actual elapsed time.
- If the process exited on its own before the budget without valid final sections, call it malformed output or process failure as appropriate, not a timeout.

If the output is missing the requested section headings but does contain parseable priority findings:
- normalize those findings into `Critical`, `High`, `Low`, and `Uncertain`
- if the normalized output still lacks a parseable `## Verdict` token, treat it as malformed and apply the retry rule above
- record in the diary that the round used normalized Codex output
- continue exactly as if Codex had emitted the requested headings

## Severity Ownership

Severity comes from `/code-review-parallel`, not from you.

Your responsibilities are:
- preserve detached Codex severity when it already uses the requested buckets
- when detached Codex emits `P0`/`P1`/`P2`/`P3`, normalize them using the mapping defined above without reinterpretation
- preserve the exact Codex verdict token (`APPROVE`, `APPROVE WITH FIXES`, or `NEEDS FIXES`)
- decide what to patch
- decide what to skip with a reason
- mark anything still unresolved after the final allowed round as `[open]`

Do not:
- rename severity levels
- collapse severity levels into custom buckets
- re-rank findings just because you disagree with the emphasis
- promote a `NEEDS FIXES` verdict to `APPROVE` because you patched everything; only Codex's own next-round verdict can end the loop

## Diary

Maintain one diary file as the source of truth:

```text
_scratch/_reviews/rocket_review_codex_<branch-safe>.md
```

Use the branch name as the identity. For the filename only, replace `/` with `-` so the file stays flat.

Create `_scratch/_reviews` if needed.

Use round-level sections, not per-finding lifecycle logs.

Required structure:

```md
# Rocket Review (Codex): <branch>

## Rocket Review Codex Round 1
### Verdict: NEEDS FIXES

### Critical
- [file:line] - description [patched] (commit abc123)

### High
- [file:line] - description [skipped: reason]

### Low
- [file:line] - description [skipped: cosmetic]

### Uncertain
- (none)
```

Rules:
- Preserve severity grouping exactly as Codex returned it.
- Keep each round self-contained.
- If a severity group has no items, write `- (none)`.
- Include the round commit hash when an item was patched in that round.
- Record the exact Codex verdict token in the round's `### Verdict:` line (`APPROVE`, `APPROVE WITH FIXES`, or `NEEDS FIXES`).
- If round 2 surfaces a new finding caused by a round 1 patch, note that explicitly in the finding text instead of inventing a new status.
- Do not claim a patch, skip, or open item unless it happened in that round.

## Final PR Comment

Post exactly one PR comment at the end, derived strictly from the diary.

Use `gh pr comment` against the current branch's PR.

Required shape:

```md
<details>
<summary>Rocket Review Summary</summary>

**Reviewer:** Codex
**Rounds:** 2
**Final Verdict:** APPROVE WITH FIXES

### Critical
- [file:line] - description [patched]

### High
- [file:line] - description [skipped: reason]

### Low
- [file:line] - description [open]

</details>
```

Rules:
- Wrap the whole PR comment body in a closed GitHub disclosure block using `<details>` and `<summary>Rocket Review Summary</summary>`.
- Do not add the `open` attribute; the disclosure must render collapsed by default.
- No claim in the PR comment may be absent from the diary.
- Preserve severity headings.
- Use `[patched]`, `[skipped: reason]`, and `[open]` exactly.
- `**Reviewer:** Codex` is always included.
- `**Final Verdict:**` must be one of:
  - `APPROVE` if the final round Codex returned exactly `APPROVE`
  - `APPROVE WITH FIXES` if the final round Codex returned `APPROVE WITH FIXES`
  - `NOT APPROVED (NEEDS FIXES after 2 rounds)` if neither round returned an approving verdict
- Do not write `APPROVE` in the final verdict unless the diary records an approving verdict from Codex in the final round.
- If only 1 round was needed (Codex approved in round 1), report `**Rounds:** 1`.
- No padding. No compliments.

## Linear Ticket Sync

Skip this step if no Linear ticket exists.

After all review rounds are done and the final PR comment is posted, update the Linear ticket description. Do not post this as a separate ticket comment.

Use the same marker-bounded managed region as `rocket_plan`:
- look for `<!-- managed:rocket-start -->` and `<!-- managed:rocket-end -->` in the description
- if both markers exist, replace everything between them (inclusive of markers)
- if markers are missing, append the managed region to the end of the description
- never touch content outside the markers
- if only one marker is found (orphaned state), treat it as missing and append a fresh managed region

When rebuilding the managed region:
- always emit both `<!-- managed:rocket-start -->` and `<!-- managed:rocket-end -->` markers
- if the implementation contract exists, include the current `## Rocket Plan Contract` block first
- if no implementation contract exists, preserve the existing `## Rocket Plan Contract` block from the current description inside the markers (do not wipe it)
- then include exactly one `Rocket Review` section
- do not create duplicate managed regions or duplicate review sections

For the review section:
- first verify the exact currently supported Linear collapsible-section syntax against official Linear editor documentation in the current session
- do not assume `>>>` or `<details>` from memory
- if collapsible syntax is clearly verified, use a collapsed section titled `Rocket Review`
- if verification is unclear, fall back to a plain `## Rocket Review` section instead of emitting broken markdown

Content requirements:
- include the final Codex verdict (`APPROVE`, `APPROVE WITH FIXES`, or `NOT APPROVED`)
- include what Codex found in each round, what was patched, what was skipped, and why skipped items were left as-is
- keep the ticket description as the source of truth for the final reviewed state

## Practical Sequence

Use this order:
1. Verify repo, branch, `gh`, `codex`, and local repo rules.
2. Make sure the review target is the current pushed branch state.
3. Resolve the PR for the current branch, creating it non-interactively if needed.
4. Check PR comments for an existing exact summary line `<summary>Rocket Review Summary</summary>`; if found, stop and report `review already complete`.
5. Build the Codex prompt with the implementation contract or fallback spec, branch, PR, repo path, and `/code-review-parallel` instruction.
6. Run round 1 with detached `codex exec --dangerously-bypass-approvals-and-sandbox` against the current pushed branch state, record the launch time, and wait up to the full `900000` ms budget before declaring timeout. Retry once on failure.
7. Parse the round 1 verdict token.
8. Update the diary for round 1 after patch/skip decisions are made.
9. If round 1 produced fixes, commit and push them, then re-verify upstream freshness.
10. If round 1 returned `APPROVE` or `APPROVE WITH FIXES`, skip to step 13.
11. Run round 2 with detached `codex exec --dangerously-bypass-approvals-and-sandbox` against the new pushed branch state. Same budget rules. Retry once on failure.
12. Parse the round 2 verdict token, update the diary, and if round 2 produced fixes, commit and push them and re-verify upstream freshness.
13. Derive one final PR comment from the diary, including `**Reviewer:** Codex`, the `**Rounds:**` count, and the `**Final Verdict:**` exactly as defined in the Final PR Comment section. Post it.
14. If a Linear ticket exists, update the ticket description with the managed contract/review tail.

## Stop Conditions

Stop immediately and report back instead of guessing if:
- you are not in a git repo/worktree
- the current branch cannot be resolved
- `gh` is unavailable or unauthenticated
- the current branch cannot be pushed or upstream cannot be made fresh
- no PR exists and deterministic `gh pr create --head ... --title ... --body-file ...` also fails
- an existing PR's head branch does not match the checked-out branch
- repo-local PR title rules cannot be satisfied from the branch commit history
- `codex` is unavailable
- both the original Codex CLI attempt and its single retry exit non-zero, are prematurely aborted, truly time out after the full `900000` ms budget, or return malformed output (including a missing or empty `## Verdict` section)
- the spec was not provided in a form you can hand to the reviewer
- the working tree contains unclear changes you cannot safely include in the review

## What This Skill Does Not Do

- It does not run more than 2 Codex review rounds.
- It does not infer approval from severity counts; only the Codex verdict token ends the loop.
- It does not collapse `APPROVE WITH FIXES` into `APPROVE` in the diary or PR comment.
- It does not silently fall back to `APPROVE` when Codex never approves; it labels the PR comment `NOT APPROVED (NEEDS FIXES after 2 rounds)` instead.
- It does not commit `_scratch` artifacts unless the user explicitly asks.
