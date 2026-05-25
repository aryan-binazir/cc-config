---
name: rocket-review-bugbot
description: Run the final review loop for a completed branch as a passive polling automation that waits for OpenAI Codex's auto-review and Cursor BugBot's auto-review on the PR. Use this whenever the user explicitly says `rocket-review-bugbot`, asks for the Codex+BugBot polling review loop, or wants the agent to ensure the current branch has a PR, then keep checking CI and PR comments every minute until both Codex and BugBot have reviewed the latest HEAD with no further actionable comments, patching or skipping comments with an audit trail and posting one final PR summary comment.
---

# Rocket Review BugBot

Use this only after implementation is complete enough for external review.

This skill is narrow on purpose:
- It does not define the implementation work.
- It does not assign or reinterpret severity.
- It does not invoke `codex exec`, `cursor-agent`, or any other reviewer CLI. The Codex GitHub integration and the Cursor BugBot GitHub App both auto-review pull requests; this skill waits for those reviews to land on the PR and acts on them.
- It does not rely on interactive PR creation.
- It is the polling counterpart of `$rocket-review-rabbit`, swapping CodeRabbit for the Codex+BugBot pair.

Your job is to take the current checked-out branch, ensure it has a PR, and then create a one-minute recurring follow-up automation that:
- watches CI on the current `HEAD`
- watches for the Codex GitHub integration's review on the current `HEAD`
- watches for the Cursor BugBot GitHub App's review on the current `HEAD`
- patches what should be patched, replies to and resolves handled threads
- pushes once per run when needed, which triggers both bots to re-review the new `HEAD`
- exits only after both bots have reviewed the latest `HEAD` with no actionable comments remaining
- posts one final PR summary comment derived from the diary

## Preconditions

Run these checks before PR resolution:

```bash
git rev-parse --is-inside-work-tree
git branch --show-current
command -v gh
gh auth status
git status -sb
```

Required conditions:
- You are inside the repo/worktree that contains the branch being reviewed.
- The intended review branch is the branch currently checked out.
- `gh` is available and authenticated.
- The Cursor BugBot GitHub App is expected to be installed on the repo. The follow-up automation will fetch BugBot's PR reviews and inline comments via `gh api`. This skill does not install, authenticate, or invoke any local BugBot CLI.
- The OpenAI Codex GitHub integration is expected to be enabled on the repo with **Automatic reviews** turned on so Codex posts a review whenever a PR is opened or updated. If automatic reviews are off, the agent should mention `@codex review` once after each push to trigger a manual review.

Before generating a PR title or PR body, read local repo rules first:
- `CLAUDE.md`
- `AGENTS.md`
- other nearby agent or workflow rules such as `.cursorrules`

Stop and report the problem if any precondition fails.

## Branch State

Both bots review the actual pushed branch state, not a local-only draft.

Before creating the follow-up automation:
- If there are review-ready local changes that belong on this branch, commit them using the repo's normal commit conventions and push them.
- If the working tree contains unrelated, ambiguous, or not-yet-ready changes, stop and ask the user instead of guessing.
- If the current branch has no upstream branch yet, push it before attempting PR creation.

After every push:
- verify that the upstream branch exists
- verify that local `HEAD` matches the upstream commit before resolving the PR or polling for reviews
- stop if upstream is stale or missing

Between automation runs:
- If the automation patches anything in a run, make one follow-up commit for that run and push it before the next run. Pushing is what triggers Codex and BugBot to re-review the new `HEAD`.
- Do not amend unless the user explicitly asks.
- Do not create extra bookkeeping commits.

## Spec Contract

This skill does not pass the spec to a reviewer CLI. Both Codex and BugBot review the diff on their own — Codex follows your `AGENTS.md` Review guidelines, and BugBot consumes the GitHub PR context directly.

What you still need to keep on disk:
- the implementation contract from `rocket_plan` persisted at `_scratch/_contracts/<branch>.md` if it exists

Contract path rules:
- Use the raw branch path, not a flattened filename.
- Example: branch `aryan-binazir/BBA-11` maps to `_scratch/_contracts/aryan-binazir/BBA-11.md`.
- Treat `_scratch/_contracts/<branch>.md` as local review state by default. Do not require it to be committed, and do not commit `_scratch` artifacts unless the user explicitly asks.

The contract is used when:
- generating the PR body if one is missing
- updating the Linear ticket at the end
- judging whether a Codex or BugBot comment is in or out of the agreed scope

If the contract file is missing and a Linear ticket exists, fetch the ticket once and use it as the secondary scope reference. If neither is available, the skill can still run but the agent must be more conservative when deciding `[skipped: out of scope]`.

## PR Resolution

The PR may or may not already exist. Resolve that non-interactively before polling begins.

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

Once the PR exists, inspect its existing comments before the automation starts.

If a comment already contains the exact summary line `<summary>Rocket Review BugBot Summary</summary>`, stop and report:

```text
review already complete
```

Do not add diary resume logic. Treat this as the only completion shortcut.

## Bot Login Discovery

This skill polls reviews posted by two GitHub apps:
- the OpenAI Codex GitHub integration
- the Cursor BugBot GitHub App

The exact bot user logins (the `user.login` GitHub returns when the bot comments) are not documented in stable form and the skill must verify them once at the start of polling rather than assuming.

On the first automation run, before deciding any "no review yet" verdict:

1. Fetch the full reviewer list for the PR:

```bash
gh api repos/<owner>/<repo>/pulls/<number>/reviews \
  --jq '[.[] | {id, user_login: .user.login, commit_id, state, submitted_at}]' \
  > _scratch/_reviews/bugbot_<branch-safe>.bot-discovery.reviews.json

gh api repos/<owner>/<repo>/pulls/<number>/comments \
  --jq '[.[] | {id, user_login: .user.login, commit_id, path, position}]' \
  > _scratch/_reviews/bugbot_<branch-safe>.bot-discovery.comments.json
```

2. Extract every unique `user_login` ending in `[bot]` from the two files.

3. Identify the two bots:
   - Treat the bot whose login matches `cursor[bot]`, `cursor-bugbot[bot]`, or `bugbot[bot]` as **BugBot**.
   - Treat the bot whose login matches `chatgpt-codex[bot]`, `codex[bot]`, `openai-codex[bot]`, or `codex-cloud[bot]` as **Codex**.
   - If exactly one candidate matches each role, persist the resolved logins to the diary and proceed.
   - If a candidate exists that does not match these patterns but is clearly the right App (for example, the only `[bot]` user that has reviewed this PR besides one obvious match), stop and ask the user to confirm the login before continuing. Do not guess.
   - If no candidate exists for one of the roles, that bot has not posted on this PR yet. Record this in the diary and proceed; the polling loop will pick the bot up the moment it posts.

4. Cache both resolved logins in the diary under a `## Bot Logins` section so subsequent runs (and the recurring automation) do not have to re-discover.

Rules:
- Do not assume one specific login string in advance. The candidate list above is a search hint, not a guarantee.
- Never hardcode a bot login into a `gh api` filter without first verifying it exists on the actual PR.

## Follow-Up Automation

Both Codex and BugBot auto-review every push to the PR. Neither returns structured severity buckets you control. Because both are asynchronous and can take time, do not keep the interactive skill blocked in a long polling loop. After the PR exists and the local branch is pushed, create a one-minute recurring follow-up automation.

Use the platform automation tool for the recurring follow-up. In Codex Desktop, minute-level recurrence is a heartbeat-style automation, not a normal detached cron. Create the follow-up with a one-minute interval, attached to this thread when available, and give it the repo path, branch, PR number, PR URL, diary path, and exact completion criteria.

There is no preset round cap. The automation runs until both bots stop adding new actionable comments on the latest pushed commit. The natural termination signal is:
- local `HEAD`, upstream `HEAD`, and PR head SHA all match
- CI for that SHA is passing, or there is no required CI to wait for
- a Codex review exists for that exact SHA
- a BugBot review exists for that exact SHA
- all inline comments from both bots tied to that SHA have been recorded in the diary as `[patched]`, `[skipped: reason]`, or `[open]`
- no actionable inline comments remain tied to that SHA from either bot
- the final PR summary comment has been posted

Only after all of those conditions are true should the automation delete or pause itself.

Codex and BugBot review PRs asynchronously. The skill reads those reviews and comments from the PR via `gh api`. There is no local CLI invocation for either bot.

### Per-run flow

Each automation run:
1. Record the current `HEAD` SHA. This is the commit the next bot reviews must cover.
2. Verify local `HEAD`, upstream `HEAD`, and PR head SHA match. If not, fetch/rebase only if the repo workflow explicitly allows it; otherwise stop and ask the user.
3. Check CI for this SHA. If CI is still pending, record the pending state in the diary and let the next automation run check again. If CI failed, record the failure and stop for the user unless the failure is clearly caused by a patch you can fix.
4. Check whether a Codex review exists whose `commit_id` matches the recorded `HEAD`. If not, record `codex: pending` for this round and let the next automation run check again. Do not treat absence as approval. If automatic Codex review is known to be off, post `@codex review` as a top-level PR comment exactly once per `HEAD` to trigger it.
5. Check whether a BugBot review exists whose `commit_id` matches the recorded `HEAD`. If not, record `bugbot: pending` for this round and let the next automation run check again. If BugBot is configured to "only when mentioned", post `bugbot run` as a top-level PR comment exactly once per `HEAD` to trigger it.
6. For each bot whose review now exists, fetch that bot's inline comments tied to this commit (see "Fetch the comments" below). Also fetch the matching `reviewThreads` via GraphQL so you have a `databaseId -> threadId` mapping for replies and resolves.
7. If both bots' reviews exist for this commit and neither has actionable inline comments on it, record the round in the diary with `(no new comments)`, post the final PR comment, update Linear if applicable, then delete or pause the automation.
8. Otherwise handle each comment (see "Handle the comments" below). Patch what should be patched.
9. If any comment was addressed by a patch in this run, create one follow-up commit and push it. Re-verify upstream freshness. The new `HEAD` becomes the target for the next automation run.
10. Reply to every handled comment thread and resolve the patched/skipped threads (see "Reply to each comment thread" below). The reply for a patched comment must reference the follow-up commit's short SHA.
11. Update the diary for this run with each comment's status and reply outcome.
12. Leave the automation active unless the completion criteria are met.

### Fetch the comments

Once a bot's review is present, capture both surfaces of its review in a single non-interactive read. Run the fetch once per bot, keyed by its resolved `user.login` from Bot Login Discovery:

```bash
BOT_LOGIN=<resolved-codex-or-bugbot-login>
ROLE=<codex|bugbot>

gh api repos/<owner>/<repo>/pulls/<number>/reviews \
  --jq "[.[] | select(.user.login == \"$BOT_LOGIN\")]" \
  > _scratch/_reviews/bugbot_<branch-safe>.round<N>.$ROLE.reviews.json

gh api repos/<owner>/<repo>/pulls/<number>/comments \
  --jq "[.[] | select(.user.login == \"$BOT_LOGIN\")]" \
  > _scratch/_reviews/bugbot_<branch-safe>.round<N>.$ROLE.comments.json
```

Rules:
- The summary review body lives in `pulls/<n>/reviews`; the inline per-line comments live in `pulls/<n>/comments`. The inline list is what drives the follow-up automation because that is where actionable findings live.
- Filter strictly to the bot's exact resolved `user.login`. Do not include human reviewers or other bots.
- For a given round, only the App comments tied to the current `HEAD` count. Older comments (already addressed or against earlier commits) are not re-raised by the App on a new commit; if they appear in your fetched JSON, scope them by their `commit_id` field to the current `HEAD`.
- Capture raw JSON per round per bot to `_scratch/_reviews/` for the diary's audit trail.
- Do not pipe through any transform that could drop or reformat comments.

### Handle the comments

After a successful fetch:
- Build one list of inline comments tied to the current `HEAD`, with each comment tagged by its source bot (`codex` or `bugbot`). Each entry should carry the comment id, thread id, file, line, source bot, and the comment body.
- Read each comment conservatively. Err toward patching clearly valid comments rather than dismissing them.
- For every comment, decide exactly one status:
  - `[patched]`
  - `[skipped: not actionable]`
  - `[skipped: reason]`
  - `[open]`
- Do not invent severity buckets for either bot's comments. Codex (in PR review mode) flags only P0/P1; BugBot focuses on real bugs/security/edge cases. Neither emits a stable severity tree, and the skill does not assign one on their behalf.
- Record the round in the diary under `## Round N` as two flat lists (one per bot) of comments with their status.

### Reply to each comment thread

Every comment you address — whether patched, skipped, or explicitly left open — must get a reply on its thread, and patched/skipped threads must be resolved. Do not leave a comment thread silent. Both bots re-review on push but neither will know whether a human/agent has decided on a thread unless the thread is resolved.

For each handled comment, in order:
1. Post a reply to the thread via `gh api repos/<owner>/<repo>/pulls/<number>/comments/<comment_id>/replies -X POST -f body=...`. Use one of these reply shapes verbatim so the audit trail is consistent:
   - `[patched]`: `Patched in <short-sha>. <one-sentence summary of the change>.`
   - `[skipped: ...]`: `Skipped: <concrete reason>.`
   - `[open]`: `Left open: <why we haven't acted yet>.`
2. For `[patched]` and `[skipped: ...]`, resolve the thread via GraphQL:
   ```bash
   gh api graphql -f query='mutation($id: ID!) { resolveReviewThread(input: { threadId: $id }) { thread { isResolved } } }' -F id=<thread_id>
   ```
   Get the thread id by querying the PR's `reviewThreads` once at the start of the round and joining each `comments.nodes[].databaseId` back to the inline comment id you handled. For `[open]`, do not resolve the thread; the open status is itself a signal that the discussion is unfinished.
3. Record the reply outcome in the diary entry for that comment (append `(replied, resolved)` or `(replied, open)`).

Reply rules:
- Replies are facts, not justifications. Do not argue with either bot; if you disagree, state the concrete reason once and resolve.
- Replies must reference the commit by short SHA when the status is `[patched]`. The SHA must match the commit that contains the patch.
- Never delete a Codex or BugBot comment thread. Resolving is the only acceptable action besides leaving it open.
- If `gh api` reply or `gh api graphql` resolve fails, stop the automation run, report the failure, and do not synthesize that the reply or resolution happened.

### Automation exit

The automation exits only after:
- CI for the current `HEAD` is passing, or there is no required CI to wait for.
- A Codex review exists for the current `HEAD`.
- A BugBot review exists for the current `HEAD`.
- The current run's fetch returned zero actionable inline comments tied to `HEAD` from either bot.
- The diary contains the terminal round with both bots showing `(no new comments)`.
- The final PR comment has been posted from the diary.
- Linear has been updated when a Linear ticket exists.

Then delete or pause the automation. Until those conditions are true, keep the one-minute follow-up active.

### Failure handling

Stop the automation run immediately and report the failure to the user on any of:
- `gh api` exits non-zero (process failure).
- The fetched JSON cannot be parsed (malformed output).
- A resolved bot login was confirmed in an earlier run but disappears from the PR reviewer list in a later run (likely uninstall or permission change); ask the user to confirm before continuing.

Do not synthesize a fake review. Do not fall back to a CLI. Do not silently treat "no findings yet" as "no findings."

Rules:
- If a comment is skipped, the diary must state the concrete reason. Vague reasons like `[skipped: not needed]` are not acceptable.
- If a comment is impossible to act on because it is underspecified, malformed, or depends on information the bot did not provide, mark it `[skipped: reason]` and explain the missing information.
- If a comment looks valid but acting on it reveals product or scope ambiguity, stop and ask the user instead of guessing.
- If a comment was already addressed by a prior automation run, mark it `[skipped: already patched in <round>]` and reference the commit. The bots usually will not re-raise once the commit it was tied to is no longer `HEAD`, but if they do, this skip handles it cleanly.

## Severity Ownership

This skill does not assign severity. Both bots emit findings as inline review comments; the skill tracks them as a flat list with status.

Your responsibilities are:
- decide what to patch
- decide what to skip with a concrete reason
- mark anything still unresolved as `[open]`
- preserve the source bot tag for each comment

Do not:
- invent severity buckets for either bot's comments
- collapse Codex and BugBot comments into a single anonymous list — keep them grouped by source bot in the diary and the final summary

## Diary

Maintain one diary file as the source of truth:

```text
_scratch/_reviews/rocket_review_bugbot_<branch-safe>.md
```

Use the branch name as the identity. For the filename only, replace `/` with `-` so the file stays flat.

Create `_scratch/_reviews` if needed.

Use round-level sections, not per-finding lifecycle logs.

Required structure:

```md
# Rocket Review BugBot: <branch>

## Bot Logins
- codex: chatgpt-codex[bot]   <!-- resolved on first run -->
- bugbot: cursor[bot]          <!-- resolved on first run -->

## Round 1
### HEAD: abc123
### CI: pending|passing|failing
### Codex review: posted | pending
### BugBot review: posted | pending

#### Codex
- [file:line] - description [patched] (commit def456)
- [file:line] - description [skipped: out of scope per contract]

#### BugBot
- [file:line] - description [patched] (commit def456)
- [file:line] - description [open]

## Round 2
### HEAD: def456
### CI: passing
### Codex review: posted
### BugBot review: posted

#### Codex
(no new comments)

#### BugBot
(no new comments)
```

Rules:
- Each `## Round N` section is one automation run, written in order as the automation runs. Include the `### HEAD: <sha>` line so it is clear which commit each bot reviewed.
- Comments are a flat list per bot, not severity-grouped. Do not invent severity buckets for them.
- For each bot, if the round produced no actionable inline comments tied to `HEAD`, write `(no new comments)` under that bot's heading.
- Keep each round self-contained.
- Include the round commit hash when an item was patched in that round.
- If a later round surfaces a new finding caused by an earlier round's patch, note that explicitly in the finding text instead of inventing a new status.
- Do not claim a patch, skip, or open item unless it happened in that round.
- The terminal round must be the run that ended the automation with both bots showing `(no new comments)`.

## Final PR Comment

Post exactly one PR comment at the end, derived strictly from the diary.

Use `gh pr comment` against the current branch's PR.

Required shape:

```md
<details>
<summary>Rocket Review BugBot Summary</summary>

**Rounds:** 2
**Codex review:** clean on <short-sha>
**BugBot review:** clean on <short-sha>

### Codex
- [file:line] - description [patched]
- [file:line] - description [skipped: reason]
- [file:line] - description [open]

### BugBot
- [file:line] - description [patched]
- [file:line] - description [skipped: reason]
- [file:line] - description [open]

</details>
```

Rules:
- Wrap the whole PR comment body in a closed GitHub disclosure block using `<details>` and `<summary>Rocket Review BugBot Summary</summary>`.
- Do not add the `open` attribute; the disclosure must render collapsed by default.
- No claim in the PR comment may be absent from the diary.
- Keep Codex findings and BugBot findings under their own `###` subheadings; do not merge them.
- Use `[patched]`, `[skipped: reason]`, and `[open]` exactly.
- `**Rounds:**` reflects the actual number of automation runs recorded in the diary, including the terminal run that produced `(no new comments)` for both bots.
- `**Codex review:** clean on <short-sha>` and `**BugBot review:** clean on <short-sha>` must reference the terminal `HEAD` for which that bot returned no actionable comments. If a bot never produced a clean run before the automation exited (rare, only on user-initiated stop), report `**Codex review:** stopped without clean run on <short-sha>` instead.
- No padding. No compliments.

## Linear Ticket Sync

Skip this step if no Linear ticket exists.

After the automation exits and the final PR comment is posted, update the Linear ticket description. Do not post this as a separate ticket comment.

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
- then include exactly one `Rocket Review BugBot` section
- do not create duplicate managed regions or duplicate review sections

For the review section:
- first verify the exact currently supported Linear collapsible-section syntax against official Linear editor documentation in the current session
- do not assume `>>>` or `<details>` from memory
- if collapsible syntax is clearly verified, use a collapsed section titled `Rocket Review BugBot`
- if verification is unclear, fall back to a plain `## Rocket Review BugBot` section instead of emitting broken markdown

Content requirements:
- include what Codex found, what BugBot found, what was patched or skipped in each automation run, and why skipped items were left as-is
- keep the ticket description as the source of truth for the final reviewed state

## Practical Sequence

Use this order:
1. Verify repo, branch, `gh`, and local repo rules.
2. Make sure the review target is the current pushed branch state.
3. Resolve the PR for the current branch, creating it non-interactively if needed.
4. Check PR comments for an existing exact summary line `<summary>Rocket Review BugBot Summary</summary>`; if found, stop and report `review already complete`.
5. On the first run, do Bot Login Discovery and persist the resolved logins to the diary.
6. Create the one-minute follow-up automation. Its prompt must include the repo path, branch, PR number, PR URL, diary path, current `HEAD`, resolved bot logins (or instructions to discover them), and the completion criteria from this skill.
7. On each automation run:
   a. Record the current `HEAD` SHA.
   b. Check CI for the recorded `HEAD`. If pending, record pending state and wait for the next run. If failed and not clearly patchable, stop and report to the user.
   c. Check `gh api repos/<owner>/<repo>/pulls/<number>/reviews` for a Codex review whose `commit_id` matches the recorded `HEAD`. If missing, record pending state and (optionally on first miss only) post `@codex review` once to trigger it.
   d. Check the same `reviews` endpoint for a BugBot review whose `commit_id` matches the recorded `HEAD`. If missing, record pending state and (optionally on first miss only) post `bugbot run` once to trigger it.
   e. For each bot whose review now exists, fetch its inline comments via `gh api`, filtered to its resolved `user.login`. Capture JSON to `_scratch/_reviews/bugbot_<branch-safe>.round<N>.<role>.{reviews,comments}.json`.
   f. Scope inline comments to the recorded `HEAD` via their `commit_id` field.
   g. If both bots have reviewed `HEAD` and neither has actionable inline comments, and CI is passing or not required, record the run in the diary with `(no new comments)` for each.
   h. Otherwise handle each comment with `[patched]`, `[skipped: ...]`, or `[open]`, record the run in the diary, commit and push any patches, re-verify upstream freshness, and wait for the next automation run with the new `HEAD`.
8. When completion criteria are met, derive one final PR comment from the diary and post it.
9. If a Linear ticket exists, update the ticket description with the managed contract/review tail.
10. Delete or pause the follow-up automation.

## Stop Conditions

Stop immediately and report back instead of guessing if:
- you are not in a git repo/worktree
- the current branch cannot be resolved
- `gh` is unavailable or unauthenticated
- the current branch cannot be pushed or upstream cannot be made fresh
- no PR exists and deterministic `gh pr create --head ... --title ... --body-file ...` also fails
- an existing PR's head branch does not match the checked-out branch
- repo-local PR title rules cannot be satisfied from the branch commit history
- any automation run's `gh api` call exits non-zero
- any fetched JSON from either bot is not parseable
- a previously-resolved bot login disappears from the PR reviewer list in a later run
- neither bot has posted any review after many automation runs and there is evidence the relevant App/integration is not installed or active on the repo; ask the user to confirm the App/integration before retrying
- the working tree contains unclear changes you cannot safely include in the review

## What This Skill Does Not Do

- It does not invoke `codex exec`, `cursor-agent`, or any other reviewer CLI.
- It does not assign severity to either bot's comments.
- It does not merge the PR.
- It does not replace repo-local rules.
- It does not hardcode bot logins; both are resolved at runtime from the PR.
- It does not commit `_scratch` artifacts unless the user explicitly asks.
- It does not declare completion until both bots have reviewed the latest `HEAD` cleanly.
