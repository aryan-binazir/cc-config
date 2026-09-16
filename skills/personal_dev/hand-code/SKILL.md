---
name: hand-code
description: Keep my hands in the language. Given a topic ("Result in Rust") or a pasted chapter that teaches one, brief me, then build the smallest sandbox with that part left as a hole. Given a ticket, reserve one slice of the plan for me. Then open it in Neovim.
disable-model-invocation: true
---

# Hand Code

Keep my hands in the language. You brief me and set the table; I type the part that exercises it.

Size the hole to fifteen minutes of typing with the brief and the test beside me. The pace is mine.

## 1. Settle the target

Offer two or three candidates, one line each on what I would build and what it teaches, in plain words for someone who has never seen the feature. We pick together, then you build.

- **Topic** (`Result in Rust`): smallest project that runs at `~/repos/_scratch/<topic>`. One feature, one file, one test that passes once I have used it properly. Tests are yours unless I say "tests too".
- **Reading** (a pasted chapter or article): a topic sandbox. Candidates come from the text's own examples, named in its words; the hole comment and brief keep its vocabulary so page and code line up. Language is the text's, or mine when it has none.
- **Ticket slice** (branch `amb/XXXX-XXX`): the plan in `_scratch/_context/<key>.md` keeps running; one slice becomes mine. Record it there as reserved for hand-coding so the plan routes around it. Whole ticket when I say so.

## 2. Build the hole

- **Test is the spec.** It pins every name, signature, and message shape.
- **Hole file** opens with a comment that teaches. First the idea in plain words: what problem it solves and how the pieces fit. Then each item to declare, in order: its signature, why it exists, how the language feature behind it works, the test that checks it. Assume I have never seen the feature. Concise, and every word explains.
- **Proof** is one command. Run it before hand-off: it fails only on the missing symbols.

## 3. Brief, then hand off

One message:

1. **Brief.** The same idea as the hole comment, plus each tool as a three-line runnable snippet and the one gotcha that bites.
2. **Hand-off.** What I am building, the idea it exercises, the proof command, the file to start in.
3. **Walk-through.** Available whenever I ask: the solution part by part against the tests.

Then run the `nvim` skill on the file to start in.
