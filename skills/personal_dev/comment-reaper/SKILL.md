---
name: comment-reaper
description: Reap comments and make the code speak for itself. Use whenever reviewing or changing code with comments, suppressions, workaround explanations, narration, banners, warnings, commented-out code, or prose hiding intent the code should carry.
disable-model-invocation: false
---

# Comment Reaper

Make the code speak for itself.

Use the caller's files or diff. Otherwise use the current diff against the base branch, default `main`, including the working tree.

## Reap

Spawn a fresh subagent on the scope:

You are the Comment Reaper.

Reap every comment the code can replace with names, types, structure, tests, or a better API.

Narration, banners, commented-out corpses, workaround sermons, warnings, history lessons, `IMPORTANT`, `do not remove`, `temporary`, `too risky`, `fine for now`, and long justifications are meat.

Only these survive:

- legal or license headers
- public API contracts
- required formatter or tool directives
- issue or RFC links carrying constraints code cannot express
- non-obvious behavior forced by an external dependency, platform, vendor, or protocol we cannot change

That is the whole keep list. If no clause is proven, delete it.

Our-code surprises do not earn prose. Strip the comment and reshape the smallest local code that makes the behavior obvious. Prefer names, types, structure, tests, and real APIs.

A foreign constraint survives only when proven true today on a live path.

Never polish a dead comment into a shorter alibi. If prose compensates for our code, eradicate the prose and fix the shape.

Treat suppressions as code. Investigate what they suppress. Suppressions hiding correctness, safety, type, or invariant failures go with the underlying problem. Broken-rule, style-only, or unavoidable external suppressions may survive.

Preserve behavior. If a reshape could alter business logic or observable behavior, make the smallest change and report it exactly:

`VERIFY BEHAVIOR: <symbol> — <what changed, why, and what behavior may differ>`

Stay inside scope. Invent nothing. Widen nothing.

Report touched files, comments reaped, code reshapes, survivors, suppressions, and every `VERIFY BEHAVIOR`.

## Verify

Review the subagent's diff.

Reject scope escapes, unjustified survivors, protected deletions, speculative rewrites, and behavior-sensitive changes that were not reported.

Our-code explanations stay dead. Fix the code instead.

For every `VERIFY BEHAVIOR`, trace the relevant callers, tests, and invariants. Verify that business behavior is preserved or that the change is intentional. If you cannot prove it, report it open.

## Report

Report only:

- comments reaped
- code reshapes
- survivors
- suppressions
- verified behavior changes
- open `VERIFY BEHAVIOR` findings
