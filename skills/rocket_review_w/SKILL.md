---
name: rocket_review_w
description: Run the final Claude review loop for a completed branch, whether or not a PR already exists. Use this whenever the user explicitly says `rocket_review`, asks for the final Claude review loop, or wants ChatGPT to ensure the current branch has a PR, have Claude review it with `code_review_parallel`, patch what should be patched, and post one final PR summary comment.
---

# Rocket Review

Use this only after implementation is complete enough for external review.

This skill is narrow on purpose:
- It does not define the implementation work.
- It does not assign or reinterpret severity.
- It does not run more than 2 external review rounds.
- It does not rely on interactive PR creation.

Your job is to take the current checked-out branch, ensure it has a PR, run one harsh Claude review and one harsh detached Cursor Agent CLI review against the supplied spec, patch what should be patched, leave a strict audit trail, and post one final PR summary comment.

## Preconditions

Run these checks before PR resolution and round 1:

```bash
git rev-parse --is-inside-work-tree
git branch --show-current
command -v gh
gh auth status
git status -sb
command -v claude
command -v cursor-agent
```

Required conditions:
- You are inside the repo/worktree that contains the branch being reviewed.
- The intended review branch is the branch currently checked out.
- `gh` is available and authenticated.
- `claude` is available on `PATH`.
- `cursor-agent` is available on `PATH`.

Before generating a PR title or PR body, read local repo rules first:
- `CLAUDE.md`
- `AGENTS.md`
- other nearby agent or workflow rules such as `.cursorrules`

Stop and report the problem if any precondition fails.

## Branch State

Each reviewer must review the actual pushed branch state, not a local-only draft.

Before round 1:
- If there are review-ready local changes that belong on this branch, commit them using the repo's normal commit conventions and push them before asking Claude to review.
- If the working tree contains unrelated, ambiguous, or not-yet-ready changes, stop and ask the user instead of guessing.
- If the current branch has no upstream branch yet, push it before attempting PR creation.

After every push:
- verify that the upstream branch exists
- verify that local `HEAD` matches the upstream commit before creating a PR or calling Claude
- stop if upstream is stale or missing

After round 1:
- If you patched anything, make one follow-up commit for that round and push it before round 2.
- Do not amend unless the user explicitly asks.
- Do not create extra bookkeeping commits.

## Spec Contract

You must supply the spec to each reviewer in the prompt you construct.

Preferred spec source (in priority order):
- an implementation contract from `rocket_plan` persisted at `_scratch/_contracts/<branch>.md`
- a Jira ticket ID
- a full Jira ticket URL

Contract path rules:
- Use the raw branch path, not a flattened filename.
- Example: branch `aryan-binazir/BBA-11` maps to `_scratch/_contracts/aryan-binazir/BBA-11.md`.

The `rocket_plan` contract is the best review target because it contains:
- `Goal`
- `Accepted scope`
- `Assumptions`
- `Out of scope`
- `Validation approach`

Fallback:
- paste the full spec text verbatim into the prompt

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
- Use `gh pr create --draft --head <current-branch> --title ... --body-file ...` or an equivalent fully explicit non-interactive form.
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

If a comment already contains the exact heading `## Rocket Review Summary`, stop and report:

```text
review already complete
```

Do not add diary resume logic. Treat this as the only completion shortcut.

## Review Prompt Contract

Construct the reviewer prompt yourself. The prompt must include:
- the implementation contract or fallback spec
- the current branch name
- the PR number and PR URL
- the repo/worktree path to review
- an explicit request to review the current branch against `Goal`, `Accepted scope`, `Assumptions`, and `Validation approach`
- an explicit instruction to respect `Out of scope` items and not treat them as missing work

Require this exact output shape:
- `Critical`
- `High`
- `Low`
- `Uncertain`
- `Verdict`

Do not ask for compliments, extra summary sections, or style feedback outside that structure.

## Claude Prompt Contract

Round 1 uses Claude.

Construct the `claude --dangerously-skip-permissions -p` prompt yourself. Add an explicit instruction to use `code_review_parallel`.

## Prompt Template

Use a prompt equivalent to this:

```text
You are Claude reviewing work completed by ChatGPT.

Use your `code_review_parallel` skill for this review.

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

Review only the changes introduced on this branch. The `code_review_parallel` skill handles scoping.

Give a brutally honest review of whether the current branch satisfies the contract.

Return findings grouped exactly as:
## Critical
## High
## Low
## Uncertain
## Verdict

Within each finding, include concrete file and line references when possible.
No padding. No compliments.
```

`claude --dangerously-skip-permissions -p` may be easier to drive with a heredoc:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
claude --dangerously-skip-permissions -p "$PROMPT"
```

**important** Timeout rules:
- Allow up to the full 15-minute budget for each `claude --dangerously-skip-permissions -p` review run: `900000` ms.
- Do not stop early just because Claude has been quiet for a few minutes.
- If a review run exceeds the full `900000` ms budget, treat it as a timeout failure.

## Cursor Prompt Contract

Round 2 must use detached Cursor Agent CLI against the current pushed branch state.

Construct the `cursor-agent` prompt yourself. Add an explicit instruction to use `code_review_parallel`.

## Cursor Prompt Template

Use a prompt equivalent to this:

```text
You are Cursor CLI reviewing work completed on this branch by ChatGPT.

Use your `code_review_parallel` skill for this review.

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

Review only the changes introduced on this branch. The `code_review_parallel` skill handles scoping.

Give a brutally honest review of whether the current branch satisfies the contract.

Return findings grouped exactly as:
## Critical
## High
## Low
## Uncertain
## Verdict

Within each finding, include concrete file and line references when possible.
No padding. No compliments.
```

`cursor-agent -p --force --output-format text --workspace "<repo-path>"` may be easier to drive with a heredoc:

```bash
PROMPT=$(cat <<'EOF'
...
EOF
)
cursor-agent -p --force --output-format text --workspace "<repo-path>" "$PROMPT"
```

If detached Cursor Agent CLI returns priority-style findings instead of the requested section headings, normalize them rather than failing immediately. Treat these labels as equivalent severities:
- `P0` -> `Critical`
- `P1` -> `High`
- `P2` or `P3` -> `Low`
- findings without a usable priority, or hedged/design-observation findings without a clear severity -> `Uncertain`

If Cursor CLI returns a freeform review plus one or more `P0`/`P1`/`P2`/`P3` findings, extract those findings, map them into the standard severity buckets above, and continue the review loop.

**important** Timeout rules:
- Allow up to the full 15-minute budget for each `cursor-agent -p --force --output-format text --workspace "<repo-path>"` run: `900000` ms.
- Do not stop early just because Cursor CLI has been quiet for a few minutes.
- If a review run exceeds the full `900000` ms budget, treat it as a timeout failure.

## Detached Cursor Agent CLI Execution Contract

Round 2 timeout handling must be explicit and budget-based, not vibe-based.

- Treat detached round 2 as a single job with a total wall-clock budget of `900000` ms.
- Record the launch timestamp when `cursor-agent -p --force --output-format text --workspace "<repo-path>"` starts. Every later wait, poll, or classification decision must measure elapsed time against that original launch timestamp.
- If your tooling supports one blocking wait for `900000` ms, prefer that.
- If your tooling requires polling, recalculate `remaining_budget_ms = 900000 - elapsed_ms` after each poll and keep waiting until either:
  - the process exits, or
  - `remaining_budget_ms <= 0`
- Never use a short fixed poll schedule whose total explicit waits add up to less than `900000` ms. A sequence like `30s + 60s + 60s` is a premature abort, not a timeout policy.
- Quiet periods are normal. Progress chatter, plugin warnings, `collab: SpawnAgent`, `collab: Wait`, retry noise, or other intermediate logs are not by themselves timeout evidence and are not malformed-output evidence while the process is still running.
- Do not classify the output as malformed while `cursor-agent -p --force --output-format text --workspace "<repo-path>"` is still running and has remaining budget. Only validate the final output shape after the process exits, or after the full `900000` ms budget is actually exhausted.
- If the workflow, operator, or wrapper stops waiting before `900000` ms elapse and before `cursor-agent -p --force --output-format text --workspace "<repo-path>"` reaches a terminal result, classify that as a premature abort. Do not describe it as "Cursor CLI timed out after 15 minutes."
- If the full `900000` ms budget is exhausted and the process is still running, terminate the Cursor process gracefully, then force-kill it if it still does not exit.

## Review Loop

Maximum review rounds: 2.

Round 1:
1. Run Claude review against the current pushed branch state.
2. Read the findings conservatively. Err toward patching rather than dismissing.
3. Patch what should be fixed.
4. For each finding, decide one of:
   - `[patched]`
   - `[skipped: not actionable]`
   - `[skipped: reason]`
5. If you patched anything, create one commit for round 1 and push it.
6. Re-verify that upstream matches local `HEAD` before round 2.

Round 2:
1. Run detached Cursor Agent CLI review against the current pushed branch state, even if round 1 produced no code changes.
2. Apply the same severity-preserving review handling.
3. If you patch anything in round 2, make one commit for round 2 and push it.
4. Re-verify that upstream matches local `HEAD`.
5. Stop after round 2 even if findings remain.

After round 2:
- Any unresolved finding that still matters and is not intentionally dismissed with a reason must be marked `[open]`.
- Do not start a third round.

## Claude CLI Failure Handling

Treat the Claude invocation as failed if any of the following happens:
- the `claude --dangerously-skip-permissions -p` command exits non-zero
- the command exceeds the full `900000` ms budget
- the output does not contain the expected `Critical`, `High`, `Low`, `Uncertain`, and `Verdict` sections

If the invocation fails:
- stop the review loop immediately
- do not guess at missing structure
- report the raw Claude output and failure mode to the user
- do not write a synthesized diary entry pretending the review succeeded

## Cursor CLI Failure Handling

Distinguish these failure modes precisely:
- `premature abort`: the workflow, operator, or wrapper stopped waiting before the full `900000` ms budget elapsed and before `cursor-agent -p --force --output-format text --workspace "<repo-path>"` produced a terminal result
- `timeout`: `cursor-agent -p --force --output-format text --workspace "<repo-path>"` was still running after the full `900000` ms budget elapsed
- `process failure`: `cursor-agent -p --force --output-format text --workspace "<repo-path>"` exited non-zero
- `malformed output`: `cursor-agent -p --force --output-format text --workspace "<repo-path>"` exited within budget, but the final collected output contains neither the expected `Critical`, `High`, `Low`, `Uncertain`, and `Verdict` sections nor any parsable `P0`/`P1`/`P2`/`P3` findings that can be normalized into those sections

Output handling rules:
- Capture the complete detached Cursor Agent CLI output for the whole run, including progress chatter and the eventual final answer.
- If progress logs appear before the final answer, ignore that leading noise and extract the final structured review block from the completed output.
- If Cursor CLI exits successfully with a freeform review plus parseable `P0`/`P1`/`P2`/`P3` findings, normalize those findings instead of failing on formatting.
- Do not call a run malformed just because early/intermediate output lacks the required headings.

If the invocation fails:
- stop the review loop immediately
- do not guess at missing structure
- report the raw Cursor CLI output, exact failure mode, and actual elapsed time to the user
- do not write a synthesized diary entry pretending the review succeeded

Failure wording rules:
- Only use timeout language if the run really consumed the full `900000` ms budget.
- If the run stopped earlier than that, say it was stopped early or prematurely aborted, and include the actual elapsed time.
- If the process exited on its own before the budget without valid final sections, call it malformed output or process failure as appropriate, not a timeout.

If the output is missing the requested section headings but does contain parseable priority findings:
- normalize those findings into `Critical`, `High`, `Low`, and `Uncertain`
- derive the round verdict conservatively from the normalized findings
- record in the diary that round 2 used normalized Cursor CLI output
- continue exactly as if Cursor CLI had emitted the requested headings

## Severity Ownership

Severity comes from `code_review_parallel`, not from you.

Your responsibilities are:
- preserve Claude's severity buckets exactly
- preserve detached Cursor Agent CLI severity when it already uses the requested buckets
- when detached Cursor Agent CLI emits `P0`/`P1`/`P2`/`P3`, normalize them using the mapping defined above without reinterpretation
- decide what to patch
- decide what to skip with a reason
- mark anything still unresolved after the final allowed round as `[open]`

Do not:
- rename severity levels
- collapse severity levels into custom buckets
- re-rank findings just because you disagree with the emphasis

## Diary

Maintain one diary file as the source of truth:

```text
_scratch/_reviews/rocket_review_<branch-safe>.md
```

Use the branch name as the identity. For the filename only, replace `/` with `-` so the file stays flat.

Create `_scratch/_reviews` if needed.

Use round-level sections, not per-finding lifecycle logs.

Required structure:

```md
# Rocket Review: <branch>

## Rocket Review Round 1
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
- Preserve severity grouping exactly as the reviewer returned it.
- Keep each round self-contained.
- If a severity group has no items, write `- (none)`.
- Include the round commit hash when an item was patched in that round.
- If round 2 surfaces a new finding caused by a round 1 patch, note that explicitly in the finding text instead of inventing a new status.
- Do not claim a patch, skip, or open item unless it happened in that round.

## Final PR Comment

Post exactly one PR comment at the end, derived strictly from the diary.

Use `gh pr comment` against the current branch's PR.

Required shape:

```md
## Rocket Review Summary

**Rounds:** 2
**Final Verdict:** APPROVE

### Critical
- [file:line] - description [patched]

### High
- [file:line] - description [skipped: reason]

### Low
- [file:line] - description [open]
```

Rules:
- No claim in the PR comment may be absent from the diary.
- Preserve severity headings.
- Use `[patched]`, `[skipped: reason]`, and `[open]` exactly.
- No padding. No compliments.
- If only 1 round was needed, report `**Rounds:** 1`.

## Jira Ticket Sync

Skip this step if no Jira ticket exists.

After all review rounds are done and the final PR comment is posted, update the Jira ticket description. Do not post this as a separate ticket comment.

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
- first verify the exact currently supported Jira collapsible-section syntax against official Jira editor documentation in the current session
- do not assume `>>>` or `<details>` from memory
- if collapsible syntax is clearly verified, use a collapsed section titled `Rocket Review`
- if verification is unclear, fall back to a plain `## Rocket Review` section instead of emitting broken markdown

Content requirements:
- include what each reviewer found, what was patched, what was skipped, and why skipped items were left as-is
- keep the ticket description as the source of truth for the final reviewed state

## Practical Sequence

Use this order:
1. Verify repo, branch, `gh`, `claude`, `cursor-agent`, and local repo rules.
2. Make sure the review target is the current pushed branch state.
3. Resolve the PR for the current branch, creating it non-interactively if needed.
4. Check PR comments for an existing `## Rocket Review Summary`; if found, stop and report `review already complete`.
5. Build the Claude prompt with the implementation contract or fallback spec, branch, PR, repo path, and `code_review_parallel` instruction.
6. Run round 1 with `claude --dangerously-skip-permissions -p`.
7. Update the diary for round 1 after patch/skip decisions are made.
8. If needed, commit and push round 1 fixes, then re-verify upstream freshness.
9. Run round 2 with detached `cursor-agent -p --force --output-format text --workspace "<repo-path>"` against the current pushed branch state, record the launch time, and wait up to the full `900000` ms budget before declaring timeout.
10. Update the diary for round 2.
11. If round 2 produced final fixes, commit and push them, then re-verify upstream freshness.
12. Derive one final PR comment from the diary and post it.
13. If a Jira ticket exists, update the ticket description with the managed contract/review tail.

## Stop Conditions

Stop immediately and report back instead of guessing if:
- you are not in a git repo/worktree
- the current branch cannot be resolved
- `gh` is unavailable or unauthenticated
- the current branch cannot be pushed or upstream cannot be made fresh
- no PR exists and deterministic `gh pr create --draft --head ... --title ... --body-file ...` also fails
- an existing PR's head branch does not match the checked-out branch
- repo-local PR title rules cannot be satisfied from the branch commit history
- `claude` is unavailable
- `cursor-agent` is unavailable
- the Claude CLI call exits non-zero, times out, or returns malformed output
- the Cursor CLI call exits non-zero, is prematurely aborted, truly times out after the full `900000` ms budget, or returns malformed output
- the spec was not provided in a form you can hand to the reviewer
- the working tree contains unclear changes you cannot safely include in the review
