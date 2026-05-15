---
name: rocket-review-rabbit
description: Run the final review loop for a completed branch using one or more Codex review rounds followed by a single terminal CodeRabbit App review round read from the PR. Use this whenever the user explicitly says `rocket-review-rabbit`, asks for the Codex + CodeRabbit review loop, or wants Codex to ensure the current branch has a PR, run Codex against the branch until verdict is APPROVE, then fetch the CodeRabbit App's PR review and patch or skip those findings with an audit trail, and post one final PR summary comment.
---

# Rocket Review Rabbit

Use this only after implementation is complete enough for external review.

This skill is narrow on purpose:
- It does not define the implementation work.
- It does not assign or reinterpret severity.
- It runs Codex up to 3 rounds, stopping early when Codex explicitly approves.
- It iterates with the CodeRabbit GitHub App until CodeRabbit's latest review on the current `HEAD` has no actionable findings. The CodeRabbit App auto-reviews every push, so this loop is driven by waiting for the App's review on the latest pushed commit, patching what it raises, pushing again, and re-fetching.
- It does not invoke the CodeRabbit CLI. CodeRabbit reviews are produced asynchronously on the PR by the GitHub App, not by a local binary.
- It does not rely on interactive PR creation.

Your job is to take the current checked-out branch, ensure it has a PR, run Codex against the pushed branch until Codex approves or the round cap is reached, then drive the CodeRabbit iteration loop: wait for the App's review on the latest commit (up to 15 minutes per round), fetch findings via `gh`, patch what should be patched, commit and push, and repeat until CodeRabbit's latest review on `HEAD` is clean or the CodeRabbit round cap is reached. Record every patch/skip/open decision for every round, and post one final PR summary comment.

## Preconditions

Run these checks before PR resolution and the Codex rounds:

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
- The CodeRabbit GitHub App is expected to be installed on the repo. The terminal CodeRabbit round will fetch the App's PR review via `gh api`. This skill does not install, authenticate, or invoke any local CodeRabbit CLI.

Before generating a PR title or PR body, read local repo rules first:
- `CLAUDE.md`
- `AGENTS.md`
- other nearby agent or workflow rules such as `.cursorrules`

Stop and report the problem if any precondition fails.

## Branch State

Each reviewer must review the actual pushed branch state, not a local-only draft.

Before the Codex rounds:
- If there are review-ready local changes that belong on this branch, commit them using the repo's normal commit conventions and push them before invoking Codex.
- If the working tree contains unrelated, ambiguous, or not-yet-ready changes, stop and ask the user instead of guessing.
- If the current branch has no upstream branch yet, push it before attempting PR creation.

After every push:
- verify that the upstream branch exists
- verify that local `HEAD` matches the upstream commit before creating a PR or invoking any reviewer
- stop if upstream is stale or missing

Between review rounds:
- If you patched anything in a Codex round, make one follow-up commit for that round and push it before the next Codex round.
- If you patched anything in a CodeRabbit round, make one follow-up commit for that round and push it before waiting for CodeRabbit's next App review. Pushing is what triggers the App to re-review.
- Do not amend unless the user explicitly asks.
- Do not create extra bookkeeping commits.

## Spec Contract

You must supply the spec to Codex in the prompt you construct. CodeRabbit itself does not consume the spec; Codex does.

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

When the local contract file exists, pass Codex the absolute file path in the prompt so it can open the contract directly if useful. You may also inline the contract contents; the key requirement is that Codex receives the contract explicitly rather than having to discover it.

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

Once the PR exists, inspect its existing comments before the CodeRabbit run begins.

If a comment already contains the exact summary line `<summary>Rocket Review Rabbit Summary</summary>`, stop and report:

```text
review already complete
```

Do not add diary resume logic. Treat this as the only completion shortcut.

## Codex Review (Single Round)

Codex runs exactly once, before the CodeRabbit iteration loop. It reviews the current pushed branch state against the implementation contract.

The round:
1. Run detached Codex review against the current pushed branch state. Pass it the implementation contract only.
2. Read the findings conservatively. Err toward patching rather than dismissing.
3. Patch what should be fixed.
4. For each finding, decide one of:
   - `[patched]`
   - `[skipped: not actionable]`
   - `[skipped: reason]`
   - `[open]`
5. If you patched anything, create one commit for that round and push it.
6. Re-verify that upstream matches local `HEAD` before proceeding.
7. Update the diary for the round.

Stop conditions:
- Codex round completes (regardless of verdict). Proceed to the CodeRabbit iteration loop. Anything Codex flagged as Critical or High that you did not patch must be recorded `[open]` so the final PR summary surfaces it.
- Any Codex failure mode (see Failure Handling below) -> stop immediately and do not proceed to the CodeRabbit iteration loop.

Do not run a second Codex round. The CodeRabbit loop that follows is the iteration phase of this skill.

## CodeRabbit Iteration Loop

The CodeRabbit GitHub App auto-reviews every push to the PR. CodeRabbit does not return structured severity buckets or APPROVE/NEEDS FIXES verdicts — it just posts comments on the PR. The skill drives a loop: wait for the App's comments on the current `HEAD`, handle each comment, commit and push patches (which triggers the App to re-review the new `HEAD`), then wait again. The loop continues until CodeRabbit posts no new actionable comments on the latest `HEAD`.

There is no preset round cap. The loop runs until CodeRabbit stops adding new comments on the latest pushed commit. The natural termination signal is the App's review pipeline finishing with no new inline comments tied to `HEAD`. Each round has a 15-minute per-round wait budget to bound runaway waits.

CodeRabbit reviews PRs asynchronously. The skill reads those comments from the PR via `gh api`. There is no local CLI invocation.

### Per-round flow

Each CodeRabbit round:
1. Record the current `HEAD` SHA. This is the commit the next App review must cover.
2. Poll for the App's review of this commit (see "Wait for the CodeRabbit App review" below).
3. Fetch the App's comments tied to this commit (see "Fetch the comments" below).
4. If the fetched review has no actionable inline comments on this commit, the loop ends. Record the round in the diary with `(no new comments)`.
5. Otherwise handle each comment (see "Handle the comments" below). Patch what should be patched.
6. If any comment was addressed by a patch in this round, create one follow-up commit and push it. Re-verify upstream freshness. The new `HEAD` becomes the target for the next round.
7. Update the diary for this round with each comment's status.
8. Loop back to step 1 with the new `HEAD`.

### Wait for the CodeRabbit App review

Before fetching for a given round, the App must have actually reviewed the current `HEAD`. Poll the PR's review list until a `coderabbitai[bot]` review appears whose `commit_id` matches the recorded `HEAD`, or 15 minutes elapse.

Polling rules:
- Per-round wait budget: 15 minutes (`900000` ms).
- Poll interval: 30 seconds.
- Each poll must call `gh api repos/<owner>/<repo>/pulls/<number>/reviews` and inspect `[].user.login`, `[].commit_id`, and `[].submitted_at`.
- Stop polling as soon as a `coderabbitai[bot]` review on the recorded `HEAD` is observed.
- If the budget elapses with no eligible review for this `HEAD`, treat it as a missing review failure mode (see Failure handling below). Do not silently proceed and do not treat absence as approval.

### Fetch the comments

Once the App review is present, capture both surfaces of the App's review in a single non-interactive read:

```bash
gh api repos/<owner>/<repo>/pulls/<number>/reviews \
  --jq '[.[] | select(.user.login == "coderabbitai[bot]")]' \
  > _scratch/_reviews/coderabbit_<branch-safe>.round<N>.reviews.json

gh api repos/<owner>/<repo>/pulls/<number>/comments \
  --jq '[.[] | select(.user.login == "coderabbitai[bot]")]' \
  > _scratch/_reviews/coderabbit_<branch-safe>.round<N>.comments.json
```

Rules:
- The summary review body lives in `pulls/<n>/reviews`; the inline per-line comments live in `pulls/<n>/comments`. The inline list is what drives the iteration loop because that is where actionable findings live.
- Filter strictly to `user.login == "coderabbitai[bot]"`. Do not include human reviewers or other bots.
- For a given round, only the App comments tied to the current `HEAD` count. Older comments (already addressed or against earlier commits) are not re-raised by the App on a new commit; if they appear in your fetched JSON, scope them by their `commit_id` field to the current `HEAD`.
- Capture raw JSON per round to `_scratch/_reviews/` for the diary's audit trail.
- Do not pipe through any transform that could drop or reformat comments.

### Handle the comments

After a successful fetch:
- Build one list of inline comments tied to the current `HEAD`. Each entry should carry file, line, and the comment body.
- Read each comment conservatively. Err toward patching clearly valid comments rather than dismissing them.
- For every CodeRabbit comment, decide exactly one status:
  - `[patched]`
  - `[skipped: not actionable]`
  - `[skipped: reason]`
  - `[open]`
- Do not invent severity buckets for CodeRabbit comments. CodeRabbit does not emit reliable severity headers, and the skill does not assign them on its behalf.
- Record the round in the diary under `## CodeRabbit Round N` as a flat list of comments with their status.

### Loop exit

The loop ends and you proceed to the final PR comment when:
- The current round's fetch returned zero actionable inline comments tied to `HEAD`. The App's review pipeline has finished with nothing more to say on this commit. Record the round with `(no new comments)` and proceed.

### Failure handling

Stop the loop immediately and report the failure to the user on any of:
- `gh api` exits non-zero (process failure).
- No `coderabbitai[bot]` review for the current `HEAD` appears within the 15-minute per-round budget (missing review). Tell the user to confirm the CodeRabbit App is installed and active on the repo before retrying.
- The fetched JSON cannot be parsed (malformed output).

Do not synthesize a fake review. Do not fall back to a CLI. Do not silently treat "no findings yet" as "no findings."

Rules:
- If a CodeRabbit comment is skipped, the diary must state the concrete reason. Vague reasons like `[skipped: not needed]` are not acceptable.
- If a CodeRabbit comment is impossible to act on because it is underspecified, malformed, or depends on information CodeRabbit did not provide, mark it `[skipped: reason]` and explain the missing information.
- If a CodeRabbit comment looks valid but acting on it reveals product or scope ambiguity, stop and ask the user instead of guessing.
- If a CodeRabbit comment was already addressed by the Codex round or by a prior CodeRabbit round, mark it `[skipped: already patched in <round>]` and reference the commit. The App usually will not re-raise once the commit it was tied to is no longer `HEAD`, but if it does, this skip handles it cleanly.

## Codex Prompt Contract

Construct the `codex exec --dangerously-bypass-approvals-and-sandbox` prompt yourself for each round. Add an explicit instruction to use `/code-review-parallel`.

The prompt must include:
- the implementation contract or fallback spec
- the current branch name
- the PR number and PR URL
- the repo/worktree path to review
- an explicit request to review the current branch against `Goal`, `Accepted scope`, `Assumptions`, and `Validation approach`
- an explicit instruction to respect `Out of scope` items and not treat them as missing work
- an explicit request to flag unnecessary complexity, non-idiomatic code, duplicate abstractions, brittle shortcuts, and simpler existing repo patterns that should have been used
- on Codex rounds 2 and 3, a short status summary of which Codex findings have already been resolved or skipped, so Codex does not re-litigate them

Do not mention CodeRabbit or any CodeRabbit findings in the Codex prompt. CodeRabbit runs after Codex and is handled separately.

Require this exact output shape:
- `Critical`
- `High`
- `Low`
- `Uncertain`
- `Verdict`

Do not ask Codex for compliments, extra summary sections, or style feedback outside that structure.

## Codex Prompt Template

Use a prompt equivalent to this for round 1:

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

Verdict must be one of: APPROVE, NEEDS FIXES.

Within each finding, include concrete file and line references when possible.
No padding. No compliments.
```

For rounds 2 and 3, also pass forward the prior round's status ledger:

```text
Prior round status:
- [file:line] - description [patched in round 1, commit abc123]
- [file:line] - description [skipped in round 1: reason]
- ...

Do not re-raise findings already marked [patched] unless the patch is demonstrably wrong. Do not re-raise [skipped] findings unless you have new information that overrides the prior skip reason.
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

Codex timeout handling must be explicit and budget-based, not vibe-based.

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

## Codex CLI Failure Handling

Distinguish these failure modes precisely:
- `premature abort`: the workflow, operator, or wrapper stopped waiting before the full `900000` ms budget elapsed and before `codex exec --dangerously-bypass-approvals-and-sandbox` produced a terminal result
- `timeout`: `codex exec --dangerously-bypass-approvals-and-sandbox` was still running after the full `900000` ms budget elapsed
- `process failure`: `codex exec --dangerously-bypass-approvals-and-sandbox` exited non-zero
- `malformed output`: `codex exec --dangerously-bypass-approvals-and-sandbox` exited within budget, but the final collected output contains neither the expected `Critical`, `High`, `Low`, `Uncertain`, and `Verdict` sections nor any parsable `P0`/`P1`/`P2`/`P3` findings that can be normalized into those sections

Output handling rules:
- Capture the complete detached Codex output for the whole run, including progress chatter and the eventual final answer.
- If progress logs appear before the final answer, ignore that leading noise and extract the final structured review block from the completed output.
- If Codex exits successfully with a freeform review plus parseable `P0`/`P1`/`P2`/`P3` findings, normalize those findings instead of failing on formatting.
- Do not call a run malformed just because early/intermediate output lacks the required headings.

If the invocation fails:
- stop the review loop immediately
- do not guess at missing structure
- report the raw Codex output, exact failure mode, and actual elapsed time to the user
- do not write a synthesized diary entry pretending the review succeeded

Failure wording rules:
- Only use timeout language if the run really consumed the full `900000` ms budget.
- If the run stopped earlier than that, say it was stopped early or prematurely aborted, and include the actual elapsed time.
- If the process exited on its own before the budget without valid final sections, call it malformed output or process failure as appropriate, not a timeout.

If the output is missing the requested section headings but does contain parseable priority findings:
- normalize those findings into `Critical`, `High`, `Low`, and `Uncertain`
- derive the round verdict conservatively from the normalized findings
- record in the diary that the round used normalized Codex output
- continue exactly as if Codex had emitted the requested headings

## Severity Ownership

Severity comes from `/code-review-parallel` (Codex), not from you. CodeRabbit comments do not carry reliable severity headers and the skill does not assign them — CodeRabbit comments are tracked as a flat list with status.

Your responsibilities are:
- preserve Codex severity when it already uses the `Critical`, `High`, `Low`, `Uncertain` buckets
- when Codex emits `P0`/`P1`/`P2`/`P3`, normalize them using the mapping defined elsewhere in this skill without reinterpretation
- decide what to patch
- decide what to skip with a reason
- mark anything still unresolved as `[open]`

Do not:
- rename Codex severity levels
- collapse severity levels into custom buckets
- re-rank findings just because you disagree with the emphasis
- invent severity buckets for CodeRabbit comments

## Diary

Maintain one diary file as the source of truth:

```text
_scratch/_reviews/rocket_review_rabbit_<branch-safe>.md
```

Use the branch name as the identity. For the filename only, replace `/` with `-` so the file stays flat.

Create `_scratch/_reviews` if needed.

Use round-level sections, not per-finding lifecycle logs.

Required structure (one Codex round, then one section per CodeRabbit iteration round):

```md
# Rocket Review Rabbit: <branch>

## Codex Round
### Verdict: NEEDS FIXES

### Critical
- [file:line] - description [patched] (commit abc123)

### High
- [file:line] - description [skipped: reason]

### Low
- [file:line] - description [skipped: cosmetic]

### Uncertain
- (none)

## CodeRabbit Round 1
### HEAD: abc123

- [file:line] - description [patched] (commit def456)
- [file:line] - description [skipped: reason]
- [file:line] - description [open]

## CodeRabbit Round 2
### HEAD: def456

(no new comments)
```

Rules:
- The `## Codex Round` section is the single Codex review round, with Codex's severity grouping preserved.
- Each `## CodeRabbit Round N` section is one CodeRabbit App review round, written in order as the CodeRabbit iteration loop runs. Include the `### HEAD: <sha>` line so it is clear which commit the App reviewed.
- CodeRabbit rounds are a flat list of comments, not severity-grouped. Do not invent severity buckets for them.
- For Codex, preserve severity grouping exactly as Codex returned it, after the documented severity mapping. If a severity group has no items, write `- (none)`.
- For CodeRabbit, if the round produced no actionable inline comments tied to `HEAD`, write `(no new comments)` and proceed.
- Keep each round self-contained.
- Include the round commit hash when an item was patched in that round.
- If a later round surfaces a new finding caused by an earlier round's patch, note that explicitly in the finding text instead of inventing a new status.
- Do not claim a patch, skip, or open item unless it happened in that round.
- The terminal CodeRabbit round must be the round that ended the loop with `(no new comments)`.

## Final PR Comment

Post exactly one PR comment at the end, derived strictly from the diary.

Use `gh pr comment` against the current branch's PR.

Required shape:

```md
<details>
<summary>Rocket Review Rabbit Summary</summary>

**Codex rounds:** 1
**CodeRabbit rounds:** 2

### Codex
#### Critical
- [file:line] - description [patched]

#### High
- [file:line] - description [skipped: reason]

#### Low
- [file:line] - description [open]

### CodeRabbit
- [file:line] - description [patched]
- [file:line] - description [skipped: reason]
- [file:line] - description [open]

</details>
```

Rules:
- Wrap the whole PR comment body in a closed GitHub disclosure block using `<details>` and `<summary>Rocket Review Rabbit Summary</summary>`.
- Do not add the `open` attribute; the disclosure must render collapsed by default.
- No claim in the PR comment may be absent from the diary.
- Preserve Codex's severity headings.
- CodeRabbit comments render as a flat list, no severity headings.
- Use `[patched]`, `[skipped: reason]`, and `[open]` exactly.
- No padding. No compliments.
- `**Codex rounds:** 1` is always literal `1`. If it is anything else, you violated the single-round Codex rule.
- `**CodeRabbit rounds:**` reflects the actual number of CodeRabbit iteration rounds executed, including the terminal round that produced `(no new comments)`.

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
- then include exactly one `Rocket Review Rabbit` section
- do not create duplicate managed regions or duplicate review sections

For the review section:
- first verify the exact currently supported Linear collapsible-section syntax against official Linear editor documentation in the current session
- do not assume `>>>` or `<details>` from memory
- if collapsible syntax is clearly verified, use a collapsed section titled `Rocket Review Rabbit`
- if verification is unclear, fall back to a plain `## Rocket Review Rabbit` section instead of emitting broken markdown

Content requirements:
- include what CodeRabbit found, what was patched or skipped in the CodeRabbit round, what Codex found in each round, what was patched or skipped in Codex rounds, and why skipped items were left as-is
- keep the ticket description as the source of truth for the final reviewed state

## Practical Sequence

Use this order:
1. Verify repo, branch, `gh`, `codex`, and local repo rules.
2. Make sure the review target is the current pushed branch state.
3. Resolve the PR for the current branch, creating it non-interactively if needed.
4. Check PR comments for an existing exact summary line `<summary>Rocket Review Rabbit Summary</summary>`; if found, stop and report `review already complete`.
5. Build the Codex prompt with the implementation contract or fallback spec, branch, PR, repo path, and `/code-review-parallel` instruction. Do not mention CodeRabbit.
6. Run the single Codex round with `codex exec --dangerously-bypass-approvals-and-sandbox`.
7. Update the diary for the Codex round after patch/skip decisions are made. If anything was patched, commit and push, then re-verify upstream freshness.
8. Enter the CodeRabbit iteration loop. For each round:
   a. Record the current `HEAD` SHA.
   b. Poll `gh api repos/<owner>/<repo>/pulls/<number>/reviews` for a `coderabbitai[bot]` review whose `commit_id` matches the recorded `HEAD`. Wait up to 15 minutes (`900000` ms) at 30-second intervals.
   c. Fetch the App's review and inline comments via `gh api`, filtered to `user.login == "coderabbitai[bot]"`. Capture JSON to `_scratch/_reviews/coderabbit_<branch-safe>.round<N>.reviews.json` and `.round<N>.comments.json`.
   d. Scope inline comments to the recorded `HEAD` via their `commit_id` field.
   e. If no actionable inline comments are tied to this `HEAD`, record the round in the diary with `(no new comments)` and exit the loop.
   f. Otherwise handle each comment with `[patched]`, `[skipped: ...]`, or `[open]`, record the round in the diary, commit and push any patches, re-verify upstream freshness, and continue the loop with the new `HEAD`.
9. Derive one final PR comment from the diary and post it.
10. If a Linear ticket exists, update the ticket description with the managed contract/review tail.

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
- a Codex round exits non-zero, is prematurely aborted, truly times out after the full `900000` ms budget, or returns malformed output
- the spec was not provided in a form you can hand to Codex
- the working tree contains unclear changes you cannot safely include in the review
- any CodeRabbit iteration round's `gh api` call exits non-zero
- no `coderabbitai[bot]` review for the current `HEAD` appears on the PR within a single round's 15-minute wait budget; ask the user to confirm the CodeRabbit App is installed and active on the repo before retrying
- any fetched CodeRabbit JSON is not parseable
