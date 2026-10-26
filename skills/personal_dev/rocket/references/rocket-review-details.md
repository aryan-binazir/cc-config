# Rocket Review Details

## Reviewer Prompts

Round 1 prompt, or an equivalent:

```text
You are <reviewer.name>, reviewing this branch with `<slash_command>`.

Target: <absolute path> · <branch> · PR #<number> <url>

Spec:
<the resolved ticket or spec text>

Judge it against Goal, Accepted scope, Assumptions, and Validation approach;
Out of scope items are intentional. Ask first whether each change is necessary
and the simplest repo-idiomatic path, naming existing patterns, helpers, or
integration points it should use; then whether it is correct.

This is the one exhaustive discovery round: inventory every issue you can
substantiate across the whole target.

Return exactly:
## Critical
## High
## Low
## Uncertain
## Verdict

Cite file:line per finding. End Verdict with one token on its own line:
- APPROVE: ready to merge as-is
- APPROVE WITH FIXES: ready once the listed fixes land
- NEEDS FIXES: needs changes and another round

Keep source and Git state as you found them; isolated temp files for
verification are fine. Report findings; the implementer patches.
```

Later rounds (see `Review Rounds` in `rocket-review`) give the reviewer every
prior round's complete output, each finding's disposition, the patch commits,
and a concise patch summary. With `/code-review-ar`, label the round
`/code-review-ar single`. Open with:

```text
Round <n>. Your earlier findings, my dispositions, and patch commits are below.
On the current pushed branch, confirm each patched finding holds, recheck open
and skipped classifications, and review code changed since round <n-1> for
regressions. Use round 1's sections and verdict tokens.
```

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

Each round:

1. Run the reviewer against the current pushed branch.
2. Validate each finding against a credible code path and the spec, then mark
   it `[patched]`, `[skipped: not actionable]`, `[skipped: reason]`,
   `[open: blocker]`, or `[open: non-blocking]`.
3. Put the round's patches in one follow-up commit, push, and re-verify
   upstream matches local `HEAD`.
4. Update the diary for that round.

When a reviewer's loop ends, record fixes pushed after its last round as
post-round branch state, not re-reviewed. After every reviewer, mark each
unresolved, unskipped finding that still matters `[open: blocker]` or
`[open: non-blocking]`.

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
- Sol - `NEEDS FIXES` - round 1
- Sol - `APPROVE WITH FIXES` - round 2

## Sol Round 1
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
- [file:line] - description [patched after Sol round 2; not re-reviewed] (commit def456; validation: <command>)

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
  exact verdict token and `round <n>` label.
- Verdicts live per round, per reviewer; there is no overall Rocket verdict.
- A fix patched after a reviewer's last round keeps `[patched]` in that round
  and also appears under `Post-Round Patches` as
  `[patched after <reviewer> round <n>; not re-reviewed]`.
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
- Sol - `NEEDS FIXES` - round 1
- Sol - `APPROVE WITH FIXES` - round 2

**Unresolved blockers:** None.
**Post-round branch state:** One accepted finding patched after Sol round 2
and validated at commit def456. Sol round 2 remains `APPROVE WITH FIXES`. Not
re-reviewed by Sol.

### Sol
#### Critical
- [file:line] - description [patched]

#### High
- [file:line] - description [skipped: reason]

#### Low
- [file:line] - description [open: non-blocking]

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
- <reviewer name> — round 1: `<verdict>`; round 2: `<verdict>`
- Post-review patch — <finding> patched after <reviewer name> round 2 at
  <commit>; not re-reviewed by <reviewer name>
- Unresolved blockers — None
```

Requirements:

- Number PR groups in task order, even for a single PR, and keep each PR's
  ledger, post-review status, and blockers inside its own group.
- Name the reviewer and round on every verdict, showing every executed round.
- A post-round patch line names the PR, reviewer, round, patch commit, and
  whether that reviewer re-reviewed it; claim reviewer approval only for
  patches the reviewer re-reviewed.
