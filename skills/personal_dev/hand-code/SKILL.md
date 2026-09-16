---
name: hand-code
description: Carve out the part worth hand-coding. Given a topic ("Result in Rust"), build the smallest runnable sandbox with that part left as a hole. Given a ticket, reserve one slice of the running plan for me. Then open it in Neovim.
disable-model-invocation: true
---

# Hand Code

I type the part worth learning. You set the table and step back.

Read the target from my words:
- **Topic** (`Result in Rust`): fresh project at `~/repos/_scratch/<topic>`, minimal init, one test file that passes only once I've used the feature properly. Tests are yours unless I say "tests too".
- **Ticket slice** (branch `amb/XXXX-XXX`): the plan in `_scratch/_context/<key>.md` keeps running as is; one slice becomes mine. I name it, or you propose the one with the most to learn. Record it there as reserved for hand-coding so the plan routes around it. Whole ticket when I say so.

One sitting, one proof command. The solution is mine to write.

Hand off in five lines: what I'm building, the one idea it exercises, the proof command, where to start. Then run the `nvim` skill on where to start.
