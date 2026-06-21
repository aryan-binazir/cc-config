# Rocket Review Details

Load only the sections needed for the active phase.

## PR Body Fallback

If repo-local rules do not define a PR body shape, use:

```md
### Problem

### Changes

### Decisions

### Testing
How it was tested, or how to test it.
```

Populate it from the implementation contract, actual code changes, and validation
that actually ran.

## Reviewer Prompt

Use a prompt equivalent to:

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

Also review implementation quality. Flag any case where the branch solved the
problem in a sloppy, overcomplicated, non-idiomatic, or brittle way. Call out
simpler existing repo patterns, helpers, abstractions, or integration points
that should have been used instead.

Review only the changes introduced on this branch. The configured slash command
handles scoping. The canonical `/code-review` command runs parallel review by
default; only `/code-review single` runs the single-pass alternative.

Give a brutally honest review of whether the current branch satisfies the
contract and whether it used the simplest repo-idiomatic implementation path.

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
Use APPROVE WITH FIXES when the branch is acceptable but you are requesting
specific fixes that the implementer should apply before merge.
Use NEEDS FIXES when the branch is not yet acceptable.

Within each finding, include concrete file and line references when possible.
No padding. No compliments.
```

## Output Normalization

If a reviewer returns priority-style findings instead of the requested headings,
normalize them:
- `P0` -> `Critical`
- `P1` -> `High`
- `P2` or `P3` -> `Low`
- no usable priority, or hedged/design observations without clear severity -> `Uncertain`

If a freeform review contains parseable `P0`/`P1`/`P2`/`P3` findings, extract and
normalize them. Do not invent verdict tokens. If the `## Verdict` section is
missing or empty after normalization, treat the output as malformed.

Parse verdicts by locating `## Verdict` and taking the last non-empty line under
that section, uppercased with surrounding whitespace and trailing punctuation
stripped. Approval requires exact `APPROVE` or `APPROVE WITH FIXES`. Any other
token, including `NEEDS FIXES` or `REJECT`, is non-approval.

Do not collapse `APPROVE WITH FIXES` into `APPROVE`. Severity counts do not end a
reviewer phase; only the verdict token does.

## Review Loop

For each configured reviewer:

1. Run round 1 against the current pushed branch state.
2. Read findings conservatively and err toward patching.
3. For each finding, decide `[patched]`, `[skipped: not actionable]`,
   `[skipped: reason]`, or `[open]`.
4. If you patched anything, create one follow-up commit for that round and push it.
5. Re-verify upstream matches local `HEAD`.
6. Update the diary for that reviewer round.
7. If the reviewer returned `APPROVE` or `APPROVE WITH FIXES`, end that reviewer
   phase only after the patch/skip/open decisions and any needed follow-up commit.
8. If the reviewer returned `NEEDS FIXES`, patched changes were pushed, and
   `max_rounds` remains, run the next round for that same reviewer.
9. Stop that reviewer phase after `max_rounds`, or earlier if no patch was made
   against a `NEEDS FIXES` result.

After all reviewer phases, mark any unresolved finding that still matters and is
not intentionally dismissed as `[open]`. Do not run reviewers outside the
selected profile and do not run extra rounds beyond `max_rounds`.

## Runner Execution

Timeout rules:
- Default timeout is `900000` ms unless profile/reviewer config sets another value.
- Record the launch timestamp when the CLI starts.
- Prefer one blocking wait for the full budget when tooling supports it.
- If polling, compute remaining budget from elapsed time and keep waiting until
  process exit or budget exhaustion.
- Do not treat progress logs, plugin warnings, retry noise, or other intermediate
  output as malformed while the process is still running.

Failure modes:
- `premature abort`: the workflow stopped waiting before the budget elapsed and
  before a terminal result.
- `timeout`: the CLI was still running after the full timeout.
- `process failure`: the CLI exited non-zero.
- `malformed output`: the CLI exited within budget but no expected sections,
  no parseable priority findings, or no verdict remained after normalization.

Retry once per failed round number using the same prompt and pushed branch state.
If retry fails, stop and report raw output, exact failure mode, and elapsed time
for both attempts. Do not consume the next round as a retry.

Capture complete CLI output, including leading chatter and final answer. Extract
the final structured review block after completion.

## Diary Format

Maintain:

```text
_scratch/_reviews/<diary_name>_<branch-safe>.md
```

Use branch name as identity and replace `/` with `-` only for the filename.
Create `_scratch/_reviews` if needed.

Use reviewer-and-round sections:

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
- Preserve severity grouping as returned or normalized.
- Keep each round self-contained.
- If a severity group has no items, write `- (none)`.
- Include the round commit hash for patched items.
- If a later round finds a new issue caused by an earlier patch, say that in the
  finding text instead of inventing a new status.
- Do not claim a patch, skip, or open item unless it happened in that round.

## Final PR Comment

Post exactly one comment at the end using `gh pr comment` against the current PR.
Derive it strictly from the diary.

Shape:

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
- Use a closed `<details>` block; do not add `open`.
- Use configured reviewer names as section headings.
- Each reviewer verdict line uses the verdict token from that reviewer's final
  round. If it ended on `NEEDS FIXES`, write
  `NOT APPROVED (NEEDS FIXES after <n> rounds)`.
- Never write `APPROVE` unless the diary records an approving verdict.
- Preserve severity headings and statuses exactly.
- No padding. No compliments.

## Linear Ticket Sync

Skip if no Linear ticket exists.

After review rounds and final PR comment, update the Linear ticket description.
Do not post a separate Linear comment.

Use the shared marker-bounded region:
- `<!-- managed:rocket-start -->`
- `<!-- managed:rocket-end -->`

If both markers exist, replace everything between them, inclusive. If markers are
missing, append a fresh region. If only one marker exists, treat it as missing.
Never touch content outside the markers.

When rebuilding:
- always emit both markers
- if an implementation contract exists, include the current `## Rocket Plan Contract`
  block first
- if no contract exists, preserve the existing contract block from the current
  description inside the markers
- then include exactly one Rocket Review section
- do not create duplicate managed regions or duplicate review sections

For the review section, verify current official Linear editor documentation for
collapsible syntax in this session. Do not assume `>>>` or `<details>` from
memory. If syntax is clearly verified, use a collapsed section titled
`Rocket Review`; otherwise use a plain `## Rocket Review` heading.

Include each reviewer's findings, patched items, skipped items with reasons,
open items, and final verdict. Keep the ticket description as the final reviewed
state.
