---
name: rocket_review
description: Run the final Claude review loop for a completed branch that already has an open PR. Use this whenever the user explicitly says `rocket_review`, asks for the final Claude review loop, or wants Codex to hand the current checked-out branch and PR to Claude for a brutally honest review using `code_review_parallel`, patch what should be patched, keep a round-based diary, and post one final PR summary comment.
---

# Rocket Review

Use this only after implementation is complete enough for external review.

This skill is narrow on purpose:
- It does not define the implementation work.
- It does not assign or reinterpret severity.
- It does not run more than 2 Claude review rounds.

Your job is to take the current checked-out branch, ensure it has an open PR (creating one if needed), ask Claude for a harsh branch review against the supplied spec, patch what should be patched, and leave a strict audit trail.

## Preconditions

Run these checks before round 1:

```bash
git rev-parse --is-inside-work-tree
git branch --show-current
gh pr view --json number,url,headRefName
git status -sb
command -v claude
```

Required conditions:
- You are inside the repo/worktree that contains the branch being reviewed.
- The intended review branch is the branch currently checked out.
- A PR exists for the current branch. If `gh pr view` fails (no PR), create one with `gh pr create` using the repo's normal PR conventions. If creation fails, stop and report the problem.
- If a PR already exists, its head branch must match the currently checked-out branch.
- `claude` is available on `PATH`.

Stop and report the problem if any other precondition fails.

## Branch State

Claude must review the actual pushed branch state, not a local-only draft.

Before round 1:
- If there are review-ready local changes that belong on this branch, commit them using the repo's normal commit conventions and push them before asking Claude to review.
- If the working tree contains unrelated, ambiguous, or not-yet-ready changes, stop and ask the user instead of guessing.

After round 1:
- If you patched anything, make one follow-up commit for that round and push it before round 2.
- Do not amend unless the user explicitly asks.
- Do not create extra bookkeeping commits.

## Spec Contract

You must supply the spec to Claude in the prompt you construct.

Preferred spec source:
- a Linear ticket ID
- a full Linear ticket URL

Fallback:
- paste the full spec text verbatim into the prompt

Do not make Claude discover the spec on its own. If you cannot supply a reliable spec, stop and ask the user.

## Claude Prompt Contract

Construct the `claude --dangerously-skip-permissions -p` prompt yourself. The prompt must include:
- the original spec
- the current branch name
- the PR number and PR URL
- the repo/worktree path Claude should review
- an explicit instruction to use `code_review_parallel`
- an explicit request for a brutally honest review of the current branch/PR against the supplied spec

Tell Claude to preserve the `code_review_parallel` output shape:
- `Critical`
- `High`
- `Low`
- `Uncertain`
- `Verdict`

Do not ask Claude for compliments, extra summary sections, or style feedback outside that structure.

## Prompt Template

Use a prompt equivalent to this:

```text
You are Claude reviewing work completed by Codex.

Use your `code_review_parallel` skill for this review.

Review target:
- Repo/worktree: <absolute path>
- Branch: <branch>
- PR: #<number> <url>

Original spec:
<Linear ticket reference or full spec text>

Review only the changes introduced on this branch. The `code_review_parallel` skill handles scoping.

Give a brutally honest review of whether the current branch satisfies the spec.

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

Round 2:
1. Run Claude review again against the updated pushed branch state if and only if round 1 produced code changes.
2. Apply the same severity-preserving review handling.
3. If you patch anything in round 2, make one commit for round 2 and push it.
4. Stop after round 2 even if findings remain.

After round 2:
- Any unresolved finding that still matters and is not intentionally dismissed with a reason must be marked `[open]`.
- Do not start a third round.

If round 1 comes back clean and you made no code changes, stop after round 1 and post the final PR comment from the diary.

## Claude CLI Failure Handling

Treat the Claude invocation as failed if any of the following happens:
- the `claude --dangerously-skip-permissions -p` command exits non-zero
- the command times out
- the output does not contain the expected `Critical`, `High`, `Low`, `Uncertain`, and `Verdict` sections

If the invocation fails:
- stop the review loop immediately
- do not guess at missing structure
- report the raw Claude output and failure mode to the user
- do not write a synthesized diary entry pretending the review succeeded

## Severity Ownership

Severity comes from `code_review_parallel`, not from you.

Your responsibilities are:
- preserve Claude's severity buckets exactly
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
- Preserve severity grouping exactly as Claude returned it.
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

## Practical Sequence

Use this order:
1. Verify repo, branch, and `claude`. Ensure a PR exists (create one if needed).
2. Make sure the review target is the current pushed branch state.
3. Build the Claude prompt with the supplied spec, branch, PR, repo path, and `code_review_parallel` instruction.
4. Run round 1 with `claude --dangerously-skip-permissions -p`.
5. Update the diary for round 1 after patch/skip decisions are made.
6. If needed, commit and push round 1 fixes.
7. Run round 2 if and only if round 1 changed code.
8. Update the diary for round 2.
9. If round 2 produced final fixes, commit and push them.
10. Derive one final PR comment from the diary and post it.

## Stop Conditions

Stop immediately and report back instead of guessing if:
- you are not in a git repo/worktree
- the current branch cannot be resolved
- no PR exists and `gh pr create` also fails
- an existing PR's head branch does not match the checked-out branch
- `claude` is unavailable
- the Claude CLI call exits non-zero, times out, or returns malformed output
- the spec was not provided in a form you can hand to Claude
- the working tree contains unclear changes you cannot safely include in the review
