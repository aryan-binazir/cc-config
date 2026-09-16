---
name: hand-code
description: Keep my hands in the language. Given a topic ("Result in Rust"), brief me, then build the smallest sandbox with that part left as a hole. Given a ticket, reserve one slice of the plan for me. Then open it in Neovim.
disable-model-invocation: true
---

# Hand Code

Keep my hands in the language. You brief me and set the table; I type the part that exercises it.

Size the hole to fifteen minutes of typing with the brief and the test beside me. The pace is mine.

## 1. Settle the target

Offer two or three candidates, one line each on the feature I would exercise. We pick together, then you build.

- **Topic** (`Result in Rust`): smallest project that runs at `~/repos/_scratch/<topic>`. One feature, one file, one test that passes once I have used it properly. Tests are yours unless I say "tests too".
- **Ticket slice** (branch `amb/XXXX-XXX`): the plan in `_scratch/_context/<key>.md` keeps running; one slice becomes mine. Record it there as reserved for hand-coding so the plan routes around it. Whole ticket when I say so.

## 2. Build the hole

- **Test is the spec.** It pins every name, signature, and message shape.
- **Hole file** opens with a comment listing what to declare, in order, each item naming the test that checks it. Exact on *what*, silent on *how*.
- **Proof** is one command. Run it before hand-off: it fails only on the missing symbols.

## 3. Brief, then hand off

One message:

1. **Brief.** The problem the idea solves in a paragraph, each tool as a three-line runnable snippet, the one gotcha that bites. Two minutes to read, and everything the hole needs.
2. **Hand-off.** What I am building, the idea it exercises, the proof command, the file to start in.
3. **Walk-through.** Available whenever I ask: the solution part by part against the tests.

Then run the `nvim` skill on the file to start in.
