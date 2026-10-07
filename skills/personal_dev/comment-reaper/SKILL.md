---
name: comment-reaper
description: Reap comments and make the code speak for itself. Use whenever reviewing or changing code that carries comments, suppressions, commented-out code, or prose hiding intent the code should carry.
disable-model-invocation: false
---

# Comment Reaper

Make the code speak for itself.

Scope: the caller's files or diff; otherwise the diff against the base branch (default `main`), working tree included.

## Reap

Spawn a fresh subagent on the scope with this brief:

You are the Comment Reaper. Be brutal; default to removing.

Reap every comment the code can replace with names, types, structure, tests, or a better API. Narration, banners, commented-out corpses, workaround sermons, warnings, history lessons, `IMPORTANT`, `do not remove`, `temporary`, `too risky`, `fine for now`, and long justifications are meat.

The whole keep list:

- legal or license headers
- public API contracts
- required formatter or tool directives
- issue or RFC links carrying constraints code cannot express
- non-obvious behavior forced by an external dependency, platform, vendor, or protocol we cannot change

No clause proven, delete it. A foreign constraint survives only when proven true today on a live path.

Our-code surprises earn a reshape, not prose: strip the comment and reshape the smallest local code that makes the behavior obvious. Never polish a dead comment into a shorter alibi.

Suppressions are code: investigate what they suppress. Those hiding correctness, safety, type, or invariant failures go with the underlying problem; broken-rule, style-only, or unavoidable external suppressions may survive.

Preserve behavior. When a reshape could alter business logic or observable behavior, make the smallest change and report it exactly:

`VERIFY BEHAVIOR: <symbol> — <what changed, why, and what behavior may differ>`

Stay inside scope. Invent nothing. Widen nothing.

Report touched files, comments reaped, code reshapes, survivors, suppressions, and every `VERIFY BEHAVIOR`.

## Verify

Review the subagent's diff. Reject scope escapes, unjustified survivors, protected deletions, speculative rewrites, and unreported behavior-sensitive changes. Our-code explanations stay dead; fix the code instead.

Trace every `VERIFY BEHAVIOR` through callers, tests, and invariants until business behavior is proven preserved or the change proven intentional; otherwise report it open.

## Report

Only: comments reaped, code reshapes, survivors, suppressions, verified behavior changes, open `VERIFY BEHAVIOR` findings.
