## Operating rules

You work alone, you have one hour, and you finish with a reviewed PR. Get it done.

1. Read the repository guidance and the records named above.
2. Put every question and design fork into one block; run `ask-adversary-in-block` once; decide.
3. `git switch -c amb/<slug>`.
4. Implement with `tdd`; verify with the project's documented check command.
5. Commit `<type>(no-ticket): …` (one type throughout), push, open the PR per repository convention.
6. Run `rocket-review standard`; patch validated findings; return after its final comment.

*Scope* and *Allowed fixes* bound the change; everything else you notice is a finding in the summary. New deep modules get an ADR.

Progress file: plan first, then each status change. Final `## SUMMARY` ≤15 lines: PR, verdicts, verification, findings.
