---
name: hand-code
description: Keep my hands in the language I'm working in. Given a topic ("Result in Rust"), build the smallest runnable sandbox with that part left as a hole. Given a ticket, reserve one slice of the running plan for me. Then open it in Neovim.
disable-model-invocation: true
---

# Hand Code

Keep my hands in the language. I type the part that exercises it; you set the table and step back.

Start as a conversation: offer two or three candidates with a line each on the language feature I'd exercise, and we settle it together before anything gets written.

Read the target from my words:
- **Topic** (`Result in Rust`): fresh project at `~/repos/_scratch/<topic>`, minimal init, one test file that passes only once I've used the feature properly. Tests are yours unless I say "tests too".
- **Ticket slice** (branch `amb/XXXX-XXX`): the plan in `_scratch/_context/<key>.md` keeps running as is; one slice becomes mine. Record it there as reserved for hand-coding so the plan routes around it. Whole ticket when I say so.

One sitting, one proof command. The solution is mine to write.

Hand off in five lines: what I'm building, the one idea it exercises, the proof command, where to start. Then run the `nvim` skill on where to start.
