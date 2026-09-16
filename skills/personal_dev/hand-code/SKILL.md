---
name: hand-code
description: Set the table for me to hand-code the part worth learning. Given a topic ("Result in Rust") or a ticket slice, build the smallest runnable scaffold with that part left as a hole, then open it in Neovim.
disable-model-invocation: true
---

# Hand Code

I type the part worth learning. You build everything around it.

Read the target from my words:
- **Topic** (`Result in Rust`): fresh project at `~/repos/_scratch/<topic>`, minimal init, one test file that passes only once I've used the feature properly.
- **Ticket slice** (branch `amb/XXXX-XXX`): chores are yours, the slice is mine. I name the slice, or you propose the one with the most to learn. Fixtures, wiring, boilerplate done; the slice a stub with a failing test. Start from `_scratch/_context/<key>.md`. Whole ticket when I say so.

Tests are yours unless I say "tests too". One sitting, one proof command. Stubs stay honest holes; the solution is mine to write.

Hand off in five lines: what I'm building, the one idea it exercises, the proof command, where to start. Then run the `nvim` skill on the stub at the hole.
