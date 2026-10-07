---
name: rocket-review
description: >-
  Final configured review loop for a completed branch: ensure a PR, run the
  profile's reviewers, patch accepted findings, post one PR summary comment.
  Use for `rocket-review [PROFILE]` or any request for the final review loop.
---

# Rocket Review

Take the checked-out branch, ensure it has a PR, run the configured review
profile against the supplied spec, patch what earns a patch, keep a strict
diary, and post exactly one final PR summary comment. The skill ends there:
implementation, merging, profile switches, and rewording severities or verdicts
belong elsewhere.

Every **stop** below means halt and report; never guess past it.

## Config

`<rocket-dir>` is the absolute path of `../rocket`, resolved from this
`SKILL.md`'s real (symlink-resolved) directory. Use it for every Rocket script
and reference.

Run `uv run --script <rocket-dir>/scripts/resolve_config.py` before choosing
reviewers. It reads `rocket.local.yaml` over `rocket.example.yaml`; once it
succeeds, its output is the only config source.

The profile is the literal `rocket-review <profile>` argument, else
`defaults.review_profile`; stop if the resolver fails or
`review_profiles.<profile>` is missing.

Invoke reviewers through exactly these runner commands:
- `claude`: `claude --permission-mode auto -p "$PROMPT"`
- `codex`: `codex --sandbox read-only --ask-for-approval on-request -c approvals_reviewer=auto_review exec "$PROMPT" < /dev/null`
- `cursor-agent`: `bash "<call-cursor-skill-dir>/scripts/call.sh" "$PROMPT"`,
  where `<call-cursor-skill-dir>` is the real directory of the `call-cursor`
  skill's `SKILL.md`. The wrapper fails closed without CLI Auto-review; stop if
  the installed CLI lacks it.

Reviewer options: `model` → the runner's `--model <model>` (Cursor: via the
wrapper); Cursor `timeout_ms` → wrapper `--timeout-ms <timeout_ms>`; Claude
`effort` → `--effort <effort>`; Codex `reasoning_effort` →
`-c model_reasoning_effort="<reasoning_effort>"`. Stop if `effort` or
`reasoning_effort` sits on the wrong runner.

## Preflight

Before PR resolution, confirm a work tree on a named branch, `gh` installed and
authed (`gh auth status`), each configured runner on `PATH` with
non-interactive auth, and the tree state (`git status -sb`). Stop on any gap.

Read repo rules (`CLAUDE.md`, `AGENTS.md`, `.cursorrules`, nearby workflow
rules) before writing a PR title or body. If repo rules keep a
`_scratch/_context` file, update it as review fixes, decisions, and final
review state change.

## Branch State

Reviewers see only pushed state.

- Before round 1, commit review-ready changes that belong on this branch, per
  repo conventions, and push; stop and ask about unrelated, ambiguous, or
  unready changes. With no upstream yet, push before PR creation.
- After every push, confirm upstream exists and matches local `HEAD`; stop if
  stale or missing.
- A round's accepted fixes land in one pushed follow-up commit of real fixes;
  amend only on request.
- Rerun a reviewer only against a newly pushed `HEAD`; with `HEAD` unchanged,
  record unresolved findings and move on.
- An approval verdict closes that reviewer, even when an accepted patch then
  moves `HEAD`.

## Spec Source

Hand every reviewer the spec directly, discovery-free. Use the first available:

1. a Linear or Jira ticket ID (the key format fits both; resolve the tracker
   with available tooling)
2. a full Linear or Jira ticket URL
3. a user-supplied markdown spec path
4. explicit fallback spec text, pasted verbatim

With no reliable spec, stop and ask. `_scratch` is local
state; commit it only on explicit request.

## PR Resolution

Resolve the PR non-interactively; stop if resolution fails.

- Existing PR: `gh pr view --json number,url,headRefName`; stop if its head
  differs from the checked-out branch.
- No PR: push and freshness-check first, then
  `gh pr create --draft --head <current-branch> --title ... --body-file ...`,
  spelling out every push/fork decision in flags. Follow repo PR title/body
  rules and stop if they can't be met; when the title derives from commit
  prefixes, take it from consistent branch commit subjects and stop if they
  disagree. Absent a repo body shape, use the fallback in
  `<rocket-dir>/references/rocket-review-details.md`. Fill the body from the
  spec, landed changes, and validation that actually ran. Then resolve the
  number/URL and verify the head branch.

## Completion Shortcut

Once the PR exists, scan its comments. If one contains the configured summary
line — `<summary>Rocket Review Summary</summary>` for
`summary_title: Rocket Review Summary` — stop and report
`review already complete`. One rocket review per PR is intentional; only the
user deleting the summary comment opens a fresh one.

## Sequence

1. Resolve the review profile.
2. Preflight.
3. Push the review target; confirm upstream matches `HEAD`.
4. Resolve or create the PR.
5. Check the completion shortcut.
6. Read `<rocket-dir>/references/rocket-review-details.md`; it governs reviewer
   prompts, output parsing, runner execution and failures, the diary, the PR
   comment, Linear sync, and the completion report.
7. Run configured reviewers in order. After each round: decide patch/skip/open,
   commit and push fixes, re-verify upstream, then update the diary with the
   exact verdict and any post-round branch state.
8. Read and follow `comment-reaper` on the diff against the base branch so the
   code speaks for itself; commit and push any clearer shape and record it as
   post-round branch state.
9. Post one final PR comment derived from the diary.
10. With a Linear ticket, sync its managed region — the sole Linear write.
11. Return the final status grouped by numbered PR, naming every reviewer
    beside each executed round's exact verdict; a short implementation or
    testing summary may follow.

## Review Rounds

Run reviewers in strict order, at most two rounds each whatever the config
says. Round 1 is the one exhaustive discovery pass over the whole pushed
branch. `APPROVE` and `APPROVE WITH FIXES` end a reviewer's rounds. Round 2
runs only after a non-approval round 1 whose accepted fixes are pushed, with
`max_rounds` above `1`: one focused follow-up that hands back the full round 1
output plus patch decisions and commit, and asks whether the fixes hold —
verification of round 1 findings plus regressions from the patches, never fresh
discovery. When the slash command is `/code-review-ar`, round 2 uses
`/code-review-ar single`, a round-scoped prompt change that leaves saved config
untouched.

Prompts carry the spec, branch, PR, repo path, and slash command (template in
details), and invite non-blocking edge cases and hardening alongside approval.
Reviewers stay read-only: prompts require preserving reviewed source and Git
state, isolated temporary fixtures and caches, and cleanup of only what they
create. You patch.

A **blocker** shows the branch cannot safely deliver the core ticket: broken
goal or acceptance behavior, a concrete realistic in-scope edge case with
plainly incorrect behavior, or a credible security, data-loss,
data-corruption, or required-path concurrency failure. Distant or speculative
edge cases, defense-in-depth, maintainability, simplification, performance
beyond expected scale, and out-of-scope improvements are non-blocking, yet
still reportable and patchable.

Patch only findings validated against a credible code path that improve the
branch; a mere possibility raised to satisfy a reviewer stays unpatched. Decide
the whole round, then batch accepted fixes into its single follow-up commit.
Preserve reviewer severity buckets, ranking, and exact verdict tokens,
normalizing only the priority labels in details. Runner failures follow
details' recovery rules; stop if a round stays unresolved after its one
clarification or retry, and record the failure exactly in the diary.

Every report and artifact shows every executed round with its exact verdict,
per reviewer. Rounds never collapse into a per-reviewer final verdict, and no
overall Rocket verdict exists. Fixes patched after an approval or a reviewer's
final round are post-round branch state, marked not re-reviewed; the recorded
verdict stands and covers only what it reviewed.

## Diary

Each run starts a fresh diary at
`_scratch/_reviews/<diary_name>_<branch-with-slashes-replaced-by-dashes>.md`,
overwriting any prior one. It is the source of truth for the PR comment; every
claim there traces back to it.
