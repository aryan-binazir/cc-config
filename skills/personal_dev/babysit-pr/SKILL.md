---
name: babysit-pr
description: >-
  Watch a PR through CI and review bots, fixing real failures and findings.
  Use when asked to babysit, watch, or monitor a PR.
---

# Babysit PR

Own one PR until its checks are green and its review comments are handled.
Merge when the request includes `send` or an explicit merge; otherwise hand
back the PR open.

## Setup

Resolve the PR with `gh pr view <pr> --json url,headRefName,headRepository`
and work in a clean checkout of its head branch, pulled to the PR head.

## Wait

Run one watcher per PR, owned by the thread that owns the PR; delegated child
tasks leave watching to their parent. Review bots get 20 minutes once the
head's checks go green.

- **T3 Code:** handle current checks and comments, call `watch_pull_request`,
  and end the turn. Each wake is news; check the PR yourself. When a wake shows
  the head green with a review bot unfinished, call `unwatch_pull_request` and
  wait out the bot window with `pr-wait`.
- **Elsewhere:** run `<skill-dir>/scripts/pr-wait <pr> --until ready`
  (absolute path) through Claude Code's Monitor tool (`timeout_ms: 1800000`,
  re-armed on expiry), or as one long command. Exit 0: green, bots done or
  timed out, no unresolved threads, or merged. Exit 1: it needs you. Restart
  it after each push, each handled exit, and each resume.

Events needing no action: end the turn silently.

## Act

Act on failing checks for the current head and on every unresolved or
unanswered comment.

- **CI failure:** reproduce it locally and fix real failures; rerun an
  infrastructure flake once and report it.
- **Findings** from bots and people: verify each against the source. Fix real
  bugs in the intended case with the smallest change, preferring deletion or
  narrowing. Reply to the rest with a written reason, resolve the thread, and
  list it as a follow-up. Read `.macroscope/check-run-agents/*.md` before
  replying to Macroscope. CodeRabbit is advisory.
- **Each fix:** verify, commit per repo conventions, push, restart the watcher.
- **Base moved onto this PR's files, BEHIND, or DIRTY:** rebase, push with
  `--force-with-lease` to this PR's branch, and wait for green again.
- **GraphQL rate limit spent:** read through REST:
  `gh api repos/<owner>/<repo>/pulls/<n>` and
  `gh api repos/<owner>/<repo>/commits/<sha>/check-runs`.

## Merge Gate

Ready means all of these hold on the latest head:

- proof covers every change in the diff
- required checks are green
- review bots finished, or their 20 minutes passed, and each finding is fixed
  or dismissed with a written reason
- required approvals are in, and no review requests changes
- the branch is up to date with its base and GitHub reports it mergeable

With merge authorized: run `gh pr ready` on a draft, `git fetch`, recheck the
gate, and merge with the repo's method (its allowed methods and recently
merged PRs show which). Merge through branch protection as configured; with a
merge queue or auto-merge, stay until GitHub reports the PR merged. When only
required human approvals remain, hand back the PR as waiting for approval.

## Ask

Ask through the harness question tool (`AskUserQuestion`,
`request_user_input`) with the PR link, the blocker, and options with your
recommendation first, when the gate cannot pass without the user, the PR was
closed or made obsolete, or the next step touches data, production, or another
branch.

## Report

PR link, final head, each gate item's state, fixes pushed, dismissed findings
with reasons, and the merge commit or the PR's open state.
