# Rocket Review Details

## PR Body Fallback

When repo rules leave the PR body shape open, use:

```md
### Problem

### Changes

### Decisions

### Testing
How it was tested, or how to test it.
```

Populate it from the spec, actual code changes, and validation that actually
ran.

## Reviewer Prompts

Round 1 prompt, or an equivalent:

```text
You are <reviewer.name>, reviewing work completed on this branch. Run the
`<slash_command>` slash command for this review.

Review target:
- Repo/worktree: <absolute path>
- Branch: <branch>
- PR: #<number> <url>

Spec:
<the resolved ticket or spec text>

Review against Goal, Accepted scope, Assumptions, and Validation approach.
Out of scope items are intentional, not missing work.

Review implementation quality too: flag sloppy, overcomplicated,
non-idiomatic, or brittle solutions, and name the simpler existing repo
patterns, helpers, abstractions, or integration points that should have been
used.

Review only this branch's changes; the slash command handles scoping. The
canonical `/code-review-ar` command runs parallel review by default; only
`/code-review-ar single` runs the single-pass alternative.

Be brutally honest about whether the branch satisfies the spec via the
simplest repo-idiomatic path.

This is the exhaustive discovery round: inspect the complete review target and
find every issue you can substantiate against the spec and quality criteria.
Do not stop after the first actionable, blocking, or high-severity finding. The
implementer gets one discovery pass from you, so return the full inventory now.

Return findings grouped exactly as:
## Critical
## High
## Low
## Uncertain
## Verdict

End the Verdict section with one exact token on its own line:
- APPROVE: ready to merge as-is
- APPROVE WITH FIXES: acceptable once the specific fixes you request are
  applied before merge
- NEEDS FIXES: not yet acceptable

Give concrete file and line references per finding where possible. No padding.
No compliments.

You are a reviewer only. Preserve reviewed source and Git state; verify with
isolated temporary fixtures and caches, cleaning up only what you created.
Report findings; the implementing agent applies fixes.
```

Round 2 (see Review Loop for when it runs) is one focused fix-verification
pass. Give the reviewer its complete round 1 output, every finding's
disposition, the patch commit, and a concise patch summary. When the slash
command is `/code-review-ar`, label this round `/code-review-ar single`:
parallel discovery suits round 1, not fix verification; config stays
unchanged. Open with:

```text
You gave me these findings in round 1. I patched the accepted findings. Are you
happy with the fixes?
```

Have the reviewer inspect the current pushed branch, verify every patched
finding, confirm skipped and open findings remain correctly classified, and
report any unresolved finding or regression the patches caused, read-only, in
round 1's sections and verdict tokens. Discovery belongs to round 1.

## Output Normalization

Normalize priority-style findings, including those parsed from a freeform
review, into the required headings:

- `P0` -> `Critical`
- `P1` -> `High`
- `P2` or `P3` -> `Low`
- no usable priority, or hedged/design observations without clear severity ->
  `Uncertain`

The verdict is the last non-empty line under `## Verdict`, uppercased, with
surrounding whitespace and trailing punctuation stripped. Verdict tokens come
only from the reviewer; a missing or empty `## Verdict` after normalization is
malformed (see Runner Execution). Only exact `APPROVE` or `APPROVE WITH FIXES`
is approval; every other token, including `NEEDS FIXES` and foreign tokens
like `REJECT`, is non-approval. Record the token exactly: `APPROVE WITH FIXES`
stays distinct from `APPROVE`.

## Review Loop

Run only the selected profile's reviewers. For each:

1. Run round 1 against the current pushed branch.
2. Validate each finding against a credible code path and the spec, then
   mark it `[patched]`, `[skipped: not actionable]`, `[skipped: reason]`,
   `[open: blocker]`, or `[open: non-blocking]`.
3. Put the round's patches in one follow-up commit, push, and re-verify
   upstream matches local `HEAD`.
4. Update the diary for that round.
5. Run round 2 only when round 1 was non-approval, its fixes are pushed, and
   `max_rounds` allows it. Otherwise the reviewer phase ends with its open
   findings; record fixes pushed after it as post-round branch state, stated
   as not re-reviewed.
6. Round 2 is the last. A reviewer runs at most twice, within `max_rounds`,
   and reruns only against a newly pushed `HEAD`.

After all reviewer phases, mark every unresolved, unskipped finding that still
matters `[open: blocker]` or `[open: non-blocking]`.

## Runner Execution

Timeouts:

- Budget: the resolved `timeout_ms` (the resolver defaults it to `1500000`).
- Record the launch timestamp when the CLI starts.
- Prefer one blocking wait for the full budget when tooling supports it; when
  polling, compute the remaining budget from elapsed time and wait for process
  exit or budget exhaustion.
- Progress logs, plugin warnings, and retry noise are normal while the process
  runs.

Failure modes:

- `premature abort`: stopped waiting before the budget elapsed and before a
  terminal result.
- `timeout`: still running after the full budget.
- `process failure`: exited non-zero.
- `malformed output`: exited within budget, but normalization left no expected
  sections, no parseable priority findings, or no verdict.

Capture complete CLI output, leading chatter included, and extract the final
structured review block after completion. Recover completed reviews from
captured output or transcripts. Ask the reviewer once to clarify a missing or
malformed verdict from its existing findings. Retry only incomplete reviews,
once, with the same prompt and pushed branch state. If still unresolved, stop
and report raw output, exact failure mode, and elapsed time for both attempts.
Recovery, clarification, and retry all stay within the same round.

## Diary Format

Maintain `_scratch/_reviews/<diary_name>_<branch-safe>.md`: the branch name is
the identity, with `/` replaced by `-` only in the filename. Organize by
reviewer and round under a compact ledger:

```md
# Rocket Review: <branch>

## Review Ledger
- Cursor - `NEEDS FIXES` - 1st round
- Cursor - `APPROVE WITH FIXES` - 2nd round
- Codex - `APPROVE` - 1st round

## Cursor Round 1
### Verdict: NEEDS FIXES

### Critical
- [file:line] - description [patched] (commit abc123)

### High
- [file:line] - description [skipped: reason]

### Low
- [file:line] - description [open: non-blocking]

### Uncertain
- (none)

## Post-Review Branch State
### Post-Round Patches
- [file:line] - description [patched after Cursor round 2; not re-reviewed] (commit def456; validation: <command>)

### Unresolved Blockers
- (none)
```

Rules:

- Keep severity grouping as returned or normalized; write `- (none)` for an
  empty group.
- Keep each round self-contained, claiming only the patches, skips, and open
  items that happened in it; patched items carry the round's commit hash.
- A new issue caused by an earlier patch says so in its finding text, under an
  existing status.
- Update the ledger after every round: one line per executed round, with its
  exact verdict token (`APPROVE`, `APPROVE WITH FIXES`, `NEEDS FIXES`) and
  label (`1st round`, `2nd round`), never collapsed to each reviewer's final
  round.
- Verdicts live per round, per reviewer; there is no overall Rocket verdict.
- A fix patched after an approving round 1 keeps `[patched]` in that round and
  also appears under `Post-Round Patches` as
  `[patched after <reviewer> round 1; not re-reviewed]`.
- End with `Post-Review Branch State` when post-round patches or unresolved
  blockers exist. Post-round patches stay apart from reviewer rounds and leave
  every recorded verdict unchanged.

## Final PR Comment

Post exactly one comment at the end with `gh pr comment` on the current PR,
derived strictly from the diary:

```md
<details>
<summary><summary_title></summary>

**Profile:** <profile>
**Rounds:** <reviewer round count>
**Review ledger:**
- Cursor - `NEEDS FIXES` - 1st round
- Cursor - `APPROVE WITH FIXES` - 2nd round
- Codex - `APPROVE` - 1st round

**Unresolved blockers:** None.
**Post-round branch state:** One accepted finding patched after Cursor round 2
and validated at commit def456. Cursor round 2 remains `APPROVE WITH FIXES`.
Not re-reviewed by Cursor.

### Cursor
#### Critical
- [file:line] - description [patched]

#### High
- [file:line] - description [skipped: reason]

#### Low
- [file:line] - description [open: non-blocking]

### Codex
#### Critical
- (none)

</details>
```

Rules:

- Keep `<details>` closed (no `open`), with configured reviewer names as
  section headings.
- Copy the review ledger, unresolved blockers, and post-round branch state
  from the diary; the ledger shows every executed round with its exact
  verdict, never a per-reviewer final verdict or an overall Rocket verdict.
- Call review complete only when every configured reviewer has a ledger line.
- A post-round patch note names the round it followed and states that the
  recorded verdict is unchanged and the reviewer did not re-review it.
- Preserve severity headings and statuses exactly. No padding. No compliments.

## User-Facing Completion Report

The final assistant response is separate from the PR comment. Group status by
numbered PR, then by configured reviewer with each executed round's exact
verdict:

```md
PR 1 — <repo or service> #<number>: <url>
- <reviewer name> — 1st round: `<verdict>`; 2nd round: `<verdict>`
- <reviewer name> — 1st round: `<verdict>`; 2nd round: `<verdict>`
- Post-review patch — <finding> patched after <reviewer name> 2nd round at
  <commit>; not re-reviewed by <reviewer name>
- Unresolved blockers — None

PR 2 — <repo or service> #<number>: <url>
- <reviewer name> — 1st round: `<verdict>`; 2nd round: `<verdict>`
- <reviewer name> — 1st round: `<verdict>`; 2nd round: `<verdict>`
- Unresolved blockers — None
```

Requirements:

- Number PR groups in task order, even for a single PR, and keep each PR's
  ledger, post-review status, and blockers inside its own group.
- Name the reviewer and round on every verdict (never a bare "final-round
  `NEEDS FIXES`"), showing every executed round and omitting only rounds that
  did not run.
- A post-round patch line names the PR, reviewer, round, patch commit, and
  whether that reviewer re-reviewed it; claim reviewer approval only for
  patches the reviewer re-reviewed.

## Linear Ticket Sync

Skip without a Linear ticket. After the review rounds and final PR comment,
update the ticket description, the sole Linear write, inside the shared
marker-bounded region:

- `<!-- managed:rocket-start -->`
- `<!-- managed:rocket-end -->`

With both markers present, replace everything between them, inclusive;
otherwise (none or one), append a fresh region. Content outside the markers
stays untouched.

When rebuilding:

- Always emit both markers.
- Lead with any existing `## Rocket Plan Contract` block from the current
  description, preserved inside the markers.
- Then include exactly one Rocket Review section, keeping one managed region.

Verify collapsible syntax (`>>>`, `<details>`) against current official Linear
editor docs in this session, not memory. If clearly verified, use a collapsed
section titled `Rocket Review`; otherwise a plain `## Rocket Review` heading.

Include each reviewer's findings, patched items, skipped items with reasons,
open items, and every round's exact verdict, plus post-round branch state
separately when applicable, with no overall Rocket verdict. The description
ends as the final reviewed state.
