---
name: rocket-review-composer
description: Run the final review loop for a completed branch using up to two Composer review rounds (via `cursor-agent -p -f`) followed by up to two Codex review rounds (via `codex exec`), each running the `/code-review` slash command and gated by an `APPROVE` verdict. Use this whenever the user explicitly says `rocket-review-composer`, asks for the Composer+Codex review loop, or wants the agent to ensure the current branch has a PR, run Composer first then Codex (max 2 rounds each, early-exit on APPROVE), patch and push between rounds, and post one final PR summary comment that reports whether each agent approved.
---

# Rocket Review Composer

Use this only after implementation is complete enough for external review.

This skill is narrow on purpose:
- It does not define the implementation work.
- It does not assign or reinterpret severity.
- It does not run more than 2 rounds per agent. Composer (via `cursor-agent`) runs up to 2 rounds, then Codex (via `codex exec`) runs up to 2 rounds.
- It does not rely on interactive PR creation.
- It exits an agent's rounds early on an approving verdict.

Your job is to take the current checked-out branch, ensure it has a PR, run Composer reviews first (up to 2 rounds), then Codex reviews (up to 2 rounds), each invoking the `/code-review` slash command and parsing the returned verdict, patch what should be patched between rounds, commit and push after every non-approving round, leave a strict audit trail, and post one final PR summary comment that surfaces whether each agent approved.

A `## Verdict` containing `APPROVE` ends that agent's loop, including conditional forms such as `APPROVE WITH FIXES`. Any other verdict (for example `NEEDS FIXES`) does not.

## Preconditions

Run these checks before PR resolution and round 1:

```bash
git rev-parse --is-inside-work-tree
git branch --show-current
command -v gh
gh auth status
git status -sb
command -v cursor-agent
command -v codex
```

Required conditions:
- You are inside the repo/worktree that contains the branch being reviewed.
- The intended review branch is the branch currently checked out.
- `gh` is available and authenticated.
- `cursor-agent` is available on `PATH` and authenticated. The skill assumes Composer is the active model in the user's Cursor account; if your account routes to a different model, set the model via `cursor-agent` configuration before running.
- `codex` is available on `PATH`.

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

Between rounds:
- If you patched anything in a round, make one follow-up commit for that round and push it before the next round (of either agent). Each subsequent round must review the new pushed `HEAD`.
- Do not amend unless the user explicitly asks.
- Do not create extra bookkeeping commits.

## Spec Contract

You must supply the spec to each reviewer in the prompt you construct.

Preferred spec source (in priority order):
- an implementation contract from `rocket_plan` persisted at `_scratch/_contracts/<branch>.md`
- a Linear ticket ID
- a full Linear ticket URL

Contract path rules:
- Use the raw branch path, not a flattened filename.
- Example: branch `aryan-binazir/BBA-11` maps to `_scratch/_contracts/aryan-binazir/BBA-11.md`.
- Treat `_scratch/_contracts/<branch>.md` as local review state by default. Do not require it to be committed, and do not commit `_scratch` artifacts unless the user explicitly asks.

Fallback:
- paste the full spec text verbatim into the prompt

When the local contract file exists, pass the reviewer its absolute file path in the prompt so it can open the contract directly if useful. You may also inline the contract contents; the key requirement is that the reviewer receives the contract explicitly rather than having to discover it.

Do not make either reviewer discover the spec on its own. If you cannot supply a reliable spec, stop and ask the user.

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

If a comment already contains the exact summary line `<summary>Rocket Review Composer Summary</summary>`, stop and report:

```text
review already complete
```

Do not add diary resume logic. Treat this as the only completion shortcut.

## Review Prompt Contract

Construct each reviewer prompt yourself. The prompt must include:
- the implementation contract or fallback spec
- the current branch name
- the PR number and PR URL
- the repo/worktree path to review
- an explicit instruction to run the `/code-review` slash command for this review (the non-parallel one)
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

## Composer (cursor-agent) invocation

Invoke each Composer round via `cursor-agent` in non-interactive print mode with the user's standard flags:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
cursor-agent -p -f "$PROMPT"
```

Do not run any other `cursor-agent` mode. The flags `-p -f` are mandatory per user convention.

Use a prompt equivalent to:

```text
You are Composer reviewing work completed on this branch.

Run the `/code-review` slash command for this review. Do not use the parallel variant.

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

Review only the changes introduced on this branch. The `/code-review` command handles scoping.

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

## Codex (codex exec) invocation

Invoke each Codex round via detached `codex exec`:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
codex exec --dangerously-bypass-approvals-and-sandbox "$PROMPT"
```

Use the same prompt as Composer above with the opening line changed to "You are Codex reviewing work completed on this branch." The rest of the prompt (including the `/code-review` slash command instruction, severity buckets, and verdict tokens) is identical.

If detached Codex returns priority-style findings instead of the requested section headings, normalize them rather than failing immediately. Treat these labels as equivalent severities:
- `P0` -> `Critical`
- `P1` -> `High`
- `P2` or `P3` -> `Low`
- findings without a usable priority, or hedged/design-observation findings without a clear severity -> `Uncertain`

## Reviewer Execution Contract

Each reviewer round, regardless of agent, must follow these rules.

Timeout:
- Allow up to the full 15-minute budget per round: `900000` ms.
- Record the launch timestamp when the CLI starts.
- If your tooling supports one blocking wait for `900000` ms, prefer that.
- If polling, recalculate `remaining_budget_ms = 900000 - elapsed_ms` after each poll and keep waiting until either the process exits or the budget is exhausted.
- Never use a short fixed poll schedule whose total explicit waits add up to less than `900000` ms.
- Quiet periods are normal. Progress chatter, plugin warnings, `collab:` events, retry noise, or other intermediate logs are not by themselves timeout evidence and are not malformed-output evidence while the process is still running.
- If the workflow stops waiting before `900000` ms elapse and before the CLI reaches a terminal result, classify that as a premature abort. Do not describe it as a timeout.

Failure modes:
- `premature abort`: the workflow stopped waiting before the full budget elapsed and before a terminal result.
- `timeout`: the CLI was still running after the full `900000` ms budget elapsed.
- `process failure`: the CLI exited non-zero.
- `malformed output`: the CLI exited within budget, but the final output contains neither the expected `Critical`, `High`, `Low`, `Uncertain`, and `Verdict` sections nor any parsable `P0`/`P1`/`P2`/`P3` findings that can be normalized, or the `Verdict` section is missing or empty after normalization.

Retry:
- A failed round is eligible for exactly one automatic retry within the same round number, using the same prompt against the same pushed branch state.
- If the retry also fails, stop the review loop immediately, report the raw output, exact failure mode, and actual elapsed time for both attempts, and do not synthesize a diary entry pretending the round succeeded.
- Each round number gets one original attempt and one retry, total. Do not consume the next round's budget as an additional retry.

Output handling:
- Capture the complete CLI output for the whole run, including progress chatter and the eventual final answer.
- If progress logs appear before the final answer, ignore that leading noise and extract the final structured review block from the completed output.
- If the CLI exits successfully with a freeform review plus parseable `P0`/`P1`/`P2`/`P3` findings, normalize those findings instead of failing on formatting.
- Do not call a run malformed just because early/intermediate output lacks the required headings.

Failure wording:
- Only use timeout language if the run really consumed the full `900000` ms budget.
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

The loop runs two agents in strict order: **Composer first, then Codex**. Each agent has a max of 2 rounds and exits early on any approving verdict.

### Phase A: Composer rounds (max 2)

For round 1, then round 2 if no approval yet:

1. Run `cursor-agent -p -f` with the prompt described above against the current pushed branch state.
2. Read the findings conservatively. Err toward patching rather than dismissing.
3. Patch what should be fixed.
4. For each finding, decide one of:
   - `[patched]`
   - `[skipped: not actionable]`
   - `[skipped: reason]`
5. If you patched anything, create one commit for this round and push it.
6. Re-verify that upstream matches local `HEAD`.
7. Record the round in the diary, including the exact verdict token.
8. Inspect the verdict:
   - If `APPROVE` or `APPROVE WITH FIXES`, exit Phase A (skip any remaining Composer rounds) and proceed to Phase B.
   - Otherwise, if this was Composer round 1, continue to Composer round 2.
   - Otherwise (Composer round 2 ended without approval), record the agent as non-approving and proceed to Phase B.

### Phase B: Codex rounds (max 2)

For round 1, then round 2 if no approval yet:

1. Run `codex exec --dangerously-bypass-approvals-and-sandbox` with the prompt described above against the current pushed branch state (which now reflects any patches landed during Phase A).
2. Read the findings conservatively. Err toward patching rather than dismissing.
3. Patch what should be fixed.
4. For each finding, decide one of `[patched]`, `[skipped: not actionable]`, or `[skipped: reason]`.
5. If you patched anything, create one commit for this round and push it.
6. Re-verify that upstream matches local `HEAD`.
7. Record the round in the diary, including the exact verdict token.
8. Inspect the verdict:
   - If `APPROVE` or `APPROVE WITH FIXES`, exit Phase B (skip any remaining Codex rounds) and proceed to the final PR comment.
   - Otherwise, if this was Codex round 1, continue to Codex round 2.
   - Otherwise (Codex round 2 ended without approval), record the agent as non-approving and proceed to the final PR comment.

### After both phases

- Any unresolved finding that still matters and is not intentionally dismissed with a reason must be marked `[open]`.
- Do not run a third round of either agent under any circumstances.
- Do not loop back to Composer once Phase B has started, even if Codex flags something Composer also flagged.

## Severity Ownership

Severity comes from `/code-review`, not from you.

Your responsibilities are:
- preserve each reviewer's severity buckets exactly when it uses the `Critical`, `High`, `Low`, `Uncertain` buckets
- when a reviewer emits `P0`/`P1`/`P2`/`P3`, normalize them using the mapping defined above without reinterpretation
- preserve the exact verdict token (`APPROVE`, `APPROVE WITH FIXES`, or `NEEDS FIXES`) per round
- decide what to patch
- decide what to skip with a reason
- mark anything still unresolved after the final allowed round as `[open]`

Do not:
- rename severity levels
- collapse severity levels into custom buckets
- re-rank findings just because you disagree with the emphasis
- promote a `NEEDS FIXES` verdict to `APPROVE` because you patched everything; only the reviewer's own next-round verdict can end its phase

## Diary

Maintain one diary file as the source of truth:

```text
_scratch/_reviews/rocket_review_composer_<branch-safe>.md
```

Use the branch name as the identity. For the filename only, replace `/` with `-` so the file stays flat.

Create `_scratch/_reviews` if needed.

Use round-level sections, not per-finding lifecycle logs.

Required structure:

```md
# Rocket Review Composer: <branch>

## Composer Round 1
### Verdict: NEEDS FIXES

### Critical
- [file:line] - description [patched] (commit abc123)

### High
- [file:line] - description [skipped: reason]

### Low
- [file:line] - description [skipped: cosmetic]

### Uncertain
- (none)

## Composer Round 2
### Verdict: APPROVE WITH FIXES
...

## Codex Round 1
### Verdict: APPROVE
...
```

Rules:
- Preserve severity grouping exactly as the reviewer returned it.
- Keep each round self-contained.
- If a severity group has no items, write `- (none)`.
- Include the round commit hash when an item was patched in that round.
- Record the exact verdict token in the round's `### Verdict:` line.
- Only include rounds that actually ran. If Composer approved in round 1, do not write a Composer Round 2 section.
- If a later round surfaces a new finding caused by an earlier round's patch, note that explicitly in the finding text instead of inventing a new status.
- Do not claim a patch, skip, or open item unless it happened in that round.

## Final PR Comment

Post exactly one PR comment at the end, derived strictly from the diary.

Use `gh pr comment` against the current branch's PR.

Required shape:

```md
<details>
<summary>Rocket Review Composer Summary</summary>

**Composer rounds:** 2
**Composer verdict:** APPROVE WITH FIXES
**Codex rounds:** 1
**Codex verdict:** APPROVE

### Composer
#### Critical
- [file:line] - description [patched]

#### High
- [file:line] - description [skipped: reason]

#### Low
- [file:line] - description [open]

### Codex
#### Critical
- (none)

#### High
- (none)

#### Low
- [file:line] - description [open]

</details>
```

Rules:
- Wrap the whole PR comment body in a closed GitHub disclosure block using `<details>` and `<summary>Rocket Review Composer Summary</summary>`.
- Do not add the `open` attribute; the disclosure must render collapsed by default.
- No claim in the PR comment may be absent from the diary.
- Preserve each agent's severity headings.
- Use `[patched]`, `[skipped: reason]`, and `[open]` exactly.
- `**Composer rounds:**` and `**Codex rounds:**` reflect the actual number of rounds that ran (1 or 2).
- `**Composer verdict:**` and `**Codex verdict:**` must be the verdict token from each agent's final round that ran. If an agent's final round was `NEEDS FIXES` after 2 rounds, report `NOT APPROVED (NEEDS FIXES after 2 rounds)` for that agent's verdict.
- Do not write `APPROVE` in a verdict unless the diary records an approving verdict from that agent in its final round.
- No padding. No compliments.

## Linear Ticket Sync

Skip this step if no Linear ticket exists.

After both phases are done and the final PR comment is posted, update the Linear ticket description. Do not post this as a separate ticket comment.

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
- then include exactly one `Rocket Review Composer` section
- do not create duplicate managed regions or duplicate review sections

For the review section:
- first verify the exact currently supported Linear collapsible-section syntax against official Linear editor documentation in the current session
- do not assume `>>>` or `<details>` from memory
- if collapsible syntax is clearly verified, use a collapsed section titled `Rocket Review Composer`
- if verification is unclear, fall back to a plain `## Rocket Review Composer` section instead of emitting broken markdown

Content requirements:
- include each agent's final verdict (`APPROVE`, `APPROVE WITH FIXES`, or `NOT APPROVED`)
- include what each agent found in each round, what was patched, what was skipped, and why skipped items were left as-is
- keep the ticket description as the source of truth for the final reviewed state

## Practical Sequence

Use this order:
1. Verify repo, branch, `gh`, `cursor-agent`, `codex`, and local repo rules.
2. Make sure the review target is the current pushed branch state.
3. Resolve the PR for the current branch, creating it non-interactively if needed.
4. Check PR comments for an existing exact summary line `<summary>Rocket Review Composer Summary</summary>`; if found, stop and report `review already complete`.
5. Build the prompt template with the implementation contract or fallback spec, branch, PR, repo path, and `/code-review` instruction.
6. Run Composer round 1 with `cursor-agent -p -f`. Retry once on failure.
7. Parse the verdict. Update the diary. If patches were made, commit and push and re-verify upstream.
8. If Composer round 1 was not an approving verdict, run Composer round 2 the same way.
9. Move to Codex round 1 with `codex exec --dangerously-bypass-approvals-and-sandbox`. Retry once on failure.
10. Parse the verdict. Update the diary. If patches were made, commit and push and re-verify upstream.
11. If Codex round 1 was not an approving verdict, run Codex round 2 the same way.
12. Derive one final PR comment from the diary, including both `Composer rounds`/`verdict` and `Codex rounds`/`verdict`. Post it.
13. If a Linear ticket exists, update the ticket description with the managed contract/review tail.

## Stop Conditions

Stop immediately and report back instead of guessing if:
- you are not in a git repo/worktree
- the current branch cannot be resolved
- `gh` is unavailable or unauthenticated
- the current branch cannot be pushed or upstream cannot be made fresh
- no PR exists and deterministic `gh pr create --head ... --title ... --body-file ...` also fails
- an existing PR's head branch does not match the checked-out branch
- repo-local PR title rules cannot be satisfied from the branch commit history
- `cursor-agent` or `codex` is unavailable
- both the original CLI attempt and its single retry exit non-zero, are prematurely aborted, truly time out after the full `900000` ms budget, or return malformed output (including a missing or empty `## Verdict` section)
- the spec was not provided in a form you can hand to the reviewer
- the working tree contains unclear changes you cannot safely include in the review

## What This Skill Does Not Do

- It does not run more than 2 rounds per agent.
- It does not start Codex rounds before Composer is done (approved or exhausted).
- It does not loop back to Composer once Codex has started.
- It does not infer approval from severity counts; only the verdict token ends an agent's phase.
- It does not collapse `APPROVE WITH FIXES` into `APPROVE` in the diary or PR comment.
- It does not silently fall back to `APPROVE` when an agent never approves; it labels that agent's verdict `NOT APPROVED (NEEDS FIXES after 2 rounds)` instead.
- It does not use the `/code-review-parallel` slash command. It uses the non-parallel `/code-review`.
- It does not commit `_scratch` artifacts unless the user explicitly asks.
