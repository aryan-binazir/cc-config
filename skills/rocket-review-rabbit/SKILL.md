---
name: rocket-review-rabbit
description: Run the final review loop for a completed branch using one detached CodeRabbit CLI review round followed by one or more Codex review rounds. Use this whenever the user explicitly says `rocket-review-rabbit`, asks for the CodeRabbit + Codex review loop, or wants Codex to ensure the current branch has a PR, run CodeRabbit CLI exactly once, patch or skip CodeRabbit findings with an audit trail, then run Codex against the updated branch until verdict is APPROVE, and post one final PR summary comment.
---

# Rocket Review Rabbit

Use this only after implementation is complete enough for external review.

This skill is narrow on purpose:
- It does not define the implementation work.
- It does not assign or reinterpret severity.
- It runs CodeRabbit CLI **exactly once** for the whole job.
- It runs Codex up to 3 rounds, stopping early when Codex explicitly approves.
- It does not rely on interactive PR creation.

Your job is to take the current checked-out branch, ensure it has a PR, run **one** detached CodeRabbit CLI review, normalize and handle those findings as the first review round, patch what should be patched, record every patch/skip/open decision, then run Codex against the updated pushed branch and the CodeRabbit ledger until Codex approves or the round cap is reached. Leave a strict audit trail and post one final PR summary comment.

## Preconditions

Run these checks before PR resolution and the CodeRabbit run:

```bash
git rev-parse --is-inside-work-tree
git branch --show-current
command -v gh
gh auth status
git status -sb
command -v codex
command -v cr || command -v coderabbit
```

Required conditions:
- You are inside the repo/worktree that contains the branch being reviewed.
- The intended review branch is the branch currently checked out.
- `gh` is available and authenticated.
- `codex` is available on `PATH`.
- The CodeRabbit CLI (`cr` or `coderabbit`) is available on `PATH` and already authenticated.

If the CodeRabbit CLI is missing, stop and tell the user to install it (`curl -fsSL https://cli.coderabbit.ai/install.sh | sh` or `brew install coderabbit`) and authenticate with `cr auth login` (or `cr auth login --agent` for OAuth in non-interactive contexts) before retrying. Do not attempt to install or authenticate it yourself.

Before generating a PR title or PR body, read local repo rules first:
- `CLAUDE.md`
- `AGENTS.md`
- other nearby agent or workflow rules such as `.cursorrules`

Stop and report the problem if any precondition fails.

## Branch State

Each reviewer must review the actual pushed branch state, not a local-only draft.

Before the CodeRabbit run:
- If there are review-ready local changes that belong on this branch, commit them using the repo's normal commit conventions and push them before invoking CodeRabbit.
- If the working tree contains unrelated, ambiguous, or not-yet-ready changes, stop and ask the user instead of guessing.
- If the current branch has no upstream branch yet, push it before attempting PR creation.

After every push:
- verify that the upstream branch exists
- verify that local `HEAD` matches the upstream commit before creating a PR or invoking any reviewer
- stop if upstream is stale or missing

Between review rounds:
- If you patched anything in the CodeRabbit round, make one follow-up commit for that round and push it before the first Codex round.
- If you patched anything in a Codex round, make one follow-up commit for that round and push it before the next Codex round.
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

## CodeRabbit Run (Exactly Once)

**This is the most important rule of this skill: CodeRabbit CLI runs exactly once per invocation of `rocket-review-rabbit`. Never re-run CodeRabbit between Codex rounds. Never re-run CodeRabbit "to double-check" after Codex patches. Once. Total.**

The CodeRabbit invocation is detached, non-interactive, and produces structured JSON that Codex will consume.

Required invocation shape:

```bash
cr review --agent --type committed --base <base-branch>
```

Rules:
- The verb is `review`. `--agent`, `--base`, and `--type` are flags on the `review` subcommand, not the top-level binary. Do not invoke `cr` bare and do not omit `review`.
- Use `--agent` to emit structured findings for agent workflows on stdout. Never use `--interactive` (full-screen TUI) and do not run `cr review` bare (it falls back to `--plain`, which is fine for humans but not what this skill consumes).
- Use `--type committed` so the review targets only the committed branch state. This skill requires the branch to be pushed first, so uncommitted changes are not in scope here. (`--type all` would also cover uncommitted work; do not use it — it muddies the diff.)
- Use `--base` set to the merge-base/target branch of the PR. Resolve it from `gh pr view --json baseRefName`, not from assumption. Fall back to `main` only if `gh` cannot supply it.
- Capture the full stdout into a file under `_scratch/_reviews/coderabbit_<branch-safe>.json`. Create `_scratch/_reviews` if needed.
- Do not pipe stdout through any transform that could drop or reformat findings.
- The CLI binary may be named `cr` or `coderabbit`. They are the same binary; prefer whichever resolved in the precondition check.

Timeout rules:
- Allow up to a 30-minute budget (`1800000` ms) for the CodeRabbit run. CodeRabbit reviews are long-running.
- Do not stop early just because CodeRabbit has been quiet for a few minutes. Progress logs are normal.
- If the run exceeds the full `1800000` ms budget, treat it as a timeout failure.

Failure handling:
- If `cr review --agent` exits non-zero, the run is a process failure.
- If the JSON output is not parseable, the run is malformed output.
- If the run was aborted before its budget was exhausted and did not exit on its own, that is a premature abort, not a timeout.
- On any failure mode: stop the review loop immediately, report the raw output and exact failure mode to the user, and do not synthesize a fake review.
- Free-tier rate limit is 3 reviews/hour. If CodeRabbit reports a rate-limit error, stop and report it to the user; do not retry on a loop.

After a successful run:
- Parse the JSON into a normalized findings list. Each finding should carry severity, file, line, and description.
- Map CodeRabbit severities into the standard buckets:
  - `critical` -> `Critical`
  - `major` / `high` -> `High`
  - `minor` / `low` / `nitpick` -> `Low`
  - anything without a clear severity or hedged design observations -> `Uncertain`
- Persist the normalized findings into the diary under the CodeRabbit round before patching or skipping anything.

## CodeRabbit Finding Handling Round

CodeRabbit findings are handled before Codex runs. Do not defer CodeRabbit decisions to Codex.

After the single CodeRabbit run:
1. Normalize CodeRabbit findings into `Critical`, `High`, `Low`, and `Uncertain` using the mapping above.
2. Read each finding conservatively. Err toward patching clearly valid findings rather than dismissing them.
3. Patch what should be fixed.
4. For every CodeRabbit finding, decide exactly one status:
   - `[patched]`
   - `[skipped: not actionable]`
   - `[skipped: reason]`
   - `[open]`
5. Record the CodeRabbit round in the diary with the original severity grouping and each finding's status.
6. If any CodeRabbit finding was patched, create one follow-up commit for the CodeRabbit round and push it.
7. Re-verify that upstream exists and local `HEAD` matches upstream before running Codex.

Rules:
- Do not run Codex before the CodeRabbit round has been recorded.
- Do not ask Codex to decide whether CodeRabbit findings should be patched before you have made your own patch/skip decision.
- If a CodeRabbit finding is skipped, the diary must state the concrete reason. Vague reasons like `[skipped: not needed]` are not acceptable.
- If a CodeRabbit finding is impossible to patch because the finding is underspecified, malformed, or depends on information CodeRabbit did not provide, mark it `[skipped: reason]` and explain the missing information.
- If a CodeRabbit finding looks valid but patching it reveals product or scope ambiguity, stop and ask the user instead of guessing.
- CodeRabbit round verdict is `NEEDS FIXES` when any finding was patched or left `[open]`; it is `APPROVE` only when there are no findings or every finding was skipped with a concrete non-actionable reason.

## Codex Iterate-Until-Approve Loop

Maximum Codex rounds: **3**. Stop earlier if Codex explicitly approves.

Codex runs after the CodeRabbit round has been patched/skipped and pushed. It reviews the current pushed branch state against the implementation contract, the CodeRabbit findings file, and the CodeRabbit round ledger. It should verify the CodeRabbit handling and perform its own review. It may re-open a skipped CodeRabbit finding only if the skip reason is demonstrably wrong or incomplete.

Each round:
1. Run detached Codex review against the current pushed branch state. Pass it the implementation contract, the CodeRabbit findings file, and the CodeRabbit round status ledger.
2. Read the findings conservatively. Err toward patching rather than dismissing.
3. Patch what should be fixed.
4. For each finding, decide one of:
   - `[patched]`
   - `[skipped: not actionable]`
   - `[skipped: reason]`
   - `[open]`
5. If you patched anything, create one commit for that round and push it.
6. Re-verify that upstream matches local `HEAD` before the next round.
7. Update the diary for the round.

Stopping conditions:
- Codex emits `## Verdict` with `APPROVE` AND the round's `## Critical` and `## High` sections are empty -> loop ends, treat the run as approved.
- 3 rounds have run -> loop ends regardless of verdict. Unresolved findings become `[open]`.
- Any Codex failure mode (see Failure Handling below) -> stop immediately.

Do not start a 4th round.

If Codex emits `APPROVE` but Critical or High findings still exist, do not stop. Treat the approval as inconsistent, override it, and continue. Codex's verdict alone is not enough; the severity buckets must also be clean.

## Codex Prompt Contract

Construct the `codex exec --dangerously-bypass-approvals-and-sandbox` prompt yourself for each round. Add an explicit instruction to use `/code-review-parallel`.

The prompt must include:
- the implementation contract or fallback spec
- the current branch name
- the PR number and PR URL
- the repo/worktree path to review
- the path to the CodeRabbit findings file (`_scratch/_reviews/coderabbit_<branch-safe>.json`) and a brief description of the severity mapping you applied
- the CodeRabbit round status ledger from the diary, including every `[patched]`, `[skipped: ...]`, and `[open]` decision
- an explicit request to review the current branch against `Goal`, `Accepted scope`, `Assumptions`, and `Validation approach`
- an explicit instruction to respect `Out of scope` items and not treat them as missing work
- an explicit request to flag unnecessary complexity, non-idiomatic code, duplicate abstractions, brittle shortcuts, and simpler existing repo patterns that should have been used
- on Codex rounds 2 and 3, a short status summary of which Codex findings have already been resolved or skipped, so Codex does not re-litigate them

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

CodeRabbit findings (already produced by a prior detached `cr review --agent --type committed --base <base>` run):
<absolute path to _scratch/_reviews/coderabbit_<branch-safe>.json>

CodeRabbit severities map: critical -> Critical, major/high -> High, minor/low/nitpick -> Low, anything ambiguous -> Uncertain.

CodeRabbit round status:
- [file:line] - description [patched in CodeRabbit round, commit abc123]
- [file:line] - description [skipped in CodeRabbit round: reason]
- [file:line] - description [open]

Verify the CodeRabbit handling. Do not re-raise CodeRabbit findings already marked [patched] unless the patch is demonstrably wrong. Do not re-raise findings marked [skipped] unless the skip reason is wrong, incomplete, or contradicted by the code.

Review against:
- Goal
- Accepted scope
- Assumptions
- Validation approach

Respect Out of scope items. Do not treat them as missing work.

Also review implementation quality. Flag any case where the branch solved the problem in a sloppy, overcomplicated, non-idiomatic, or brittle way. Call out simpler existing repo patterns, helpers, abstractions, or integration points that should have been used instead.

Return your own findings and any CodeRabbit handling issues that need more work as one unified review. Do not repeat CodeRabbit findings that were already patched or skipped with a valid reason. Review only the changes introduced on this branch. The `/code-review-parallel` command handles scoping.

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

Severity comes from CodeRabbit and from `/code-review-parallel`, not from you.

Your responsibilities are:
- preserve CodeRabbit severity using the documented mapping into `Critical`, `High`, `Low`, `Uncertain`
- preserve Codex severity when it already uses the requested buckets
- when Codex emits `P0`/`P1`/`P2`/`P3`, normalize them using the mapping defined above without reinterpretation
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
_scratch/_reviews/rocket_review_rabbit_<branch-safe>.md
```

Use the branch name as the identity. For the filename only, replace `/` with `-` so the file stays flat.

Create `_scratch/_reviews` if needed.

Use round-level sections, not per-finding lifecycle logs.

Required structure:

```md
# Rocket Review Rabbit: <branch>

## CodeRabbit Round
### Verdict: NEEDS FIXES

### Critical
- [file:line] - description [patched] (commit abc123)

### High
- [file:line] - description [skipped: reason]

### Low
- [file:line] - description [open]

### Uncertain
- (none)

## Rocket Review Rabbit Round 1
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
- The `## CodeRabbit Round` section is written once after CodeRabbit findings are normalized and handled. It must include status decisions for every CodeRabbit finding.
- Each `## Rocket Review Rabbit Round N` section is a Codex round, not the CodeRabbit round.
- Preserve severity grouping exactly as each reviewer returned it, after the documented severity mapping for CodeRabbit.
- Keep each round self-contained.
- If a severity group has no items, write `- (none)`.
- Include the round commit hash when an item was patched in that round.
- If a later round surfaces a new finding caused by an earlier round's patch, note that explicitly in the finding text instead of inventing a new status.
- Do not claim a patch, skip, or open item unless it happened in that round.

## Final PR Comment

Post exactly one PR comment at the end, derived strictly from the diary.

Use `gh pr comment` against the current branch's PR.

Required shape:

```md
<details>
<summary>Rocket Review Rabbit Summary</summary>

**CodeRabbit runs:** 1
**Codex rounds:** 2
**Final Verdict:** APPROVE

### Critical
- [file:line] - description [patched]

### High
- [file:line] - description [skipped: reason]

### Low
- [file:line] - description [open]

</details>
```

Rules:
- Wrap the whole PR comment body in a closed GitHub disclosure block using `<details>` and `<summary>Rocket Review Rabbit Summary</summary>`.
- Do not add the `open` attribute; the disclosure must render collapsed by default.
- No claim in the PR comment may be absent from the diary.
- Preserve severity headings.
- Use `[patched]`, `[skipped: reason]`, and `[open]` exactly.
- No padding. No compliments.
- `**CodeRabbit runs:** 1` is always literal `1`. If it is anything else, you violated the single-pass rule.
- `**Codex rounds:**` reflects the actual number of Codex rounds executed, between `1` and `3`.

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
1. Verify repo, branch, `gh`, `codex`, CodeRabbit CLI, and local repo rules.
2. Make sure the review target is the current pushed branch state.
3. Resolve the PR for the current branch, creating it non-interactively if needed.
4. Check PR comments for an existing exact summary line `<summary>Rocket Review Rabbit Summary</summary>`; if found, stop and report `review already complete`.
5. Resolve the PR base branch via `gh pr view --json baseRefName`.
6. Run **exactly one** detached `cr review --agent --type committed --base <base>` pass. Capture JSON to `_scratch/_reviews/coderabbit_<branch-safe>.json`. Never run CodeRabbit again in this skill invocation.
7. Normalize CodeRabbit severities, handle every CodeRabbit finding with `[patched]`, `[skipped: ...]`, or `[open]`, and record the result in the diary under `## CodeRabbit Round`.
8. If the CodeRabbit round produced patches, create one CodeRabbit follow-up commit, push it, and re-verify upstream freshness.
9. Build the Codex round 1 prompt with the implementation contract or fallback spec, branch, PR, repo path, CodeRabbit findings file path, CodeRabbit round status ledger, and `/code-review-parallel` instruction.
10. Run Codex round 1 with `codex exec --dangerously-bypass-approvals-and-sandbox`.
11. Update the diary for Codex round 1 after patch/skip decisions are made.
12. If Codex round 1 verdict is `APPROVE` and Critical and High are empty, jump to step 16.
13. If needed, commit and push Codex round 1 fixes, then re-verify upstream freshness.
14. Run Codex round 2 with the CodeRabbit status ledger and prior Codex round status ledger included. Update the diary. If round 2 satisfies the approve condition, jump to step 16.
15. Run Codex round 3 with the CodeRabbit status ledger and prior Codex round status ledger included. Update the diary. Anything still unresolved becomes `[open]`.
16. Derive one final PR comment from the diary and post it.
17. If a Linear ticket exists, update the ticket description with the managed contract/review tail.

## Stop Conditions

Stop immediately and report back instead of guessing if:
- you are not in a git repo/worktree
- the current branch cannot be resolved
- `gh` is unavailable or unauthenticated
- the current branch cannot be pushed or upstream cannot be made fresh
- no PR exists and deterministic `gh pr create --head ... --title ... --body-file ...` also fails
- an existing PR's head branch does not match the checked-out branch
- repo-local PR title rules cannot be satisfied from the branch commit history
- the CodeRabbit CLI is unavailable or unauthenticated
- the CodeRabbit run exits non-zero, is rate-limited, times out after the full `1800000` ms budget, is prematurely aborted, or returns unparseable JSON
- `codex` is unavailable
- a Codex round exits non-zero, is prematurely aborted, truly times out after the full `900000` ms budget, or returns malformed output
- the spec was not provided in a form you can hand to Codex
- the working tree contains unclear changes you cannot safely include in the review
