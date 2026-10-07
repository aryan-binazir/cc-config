---
name: pr-explainer
description: Explain what the current PR changes, why it exists, and how it fits the larger system, in plain language. Use when the user wants to understand a PR.
---

# PR Explainer

Explain the current pull request to a reader starting cold. Orient them in the system first; show code and numbers last.

## Workflow

1. Read the PR, its ticket or stated goal, and the diff.
2. Inspect enough surrounding code to place the change in the system.
3. Explain in this order:
   - **System context.** The larger goal (epic, feature set, or workflow) and the concept this PR touches, told without file names, line counts, or diff stats.
   - **The ticket's goal** and how it advances that larger goal.
   - **What the PR changes** and how the important pieces work together, grouped by purpose.
   - **Verification, limitations, and intentionally unchanged behavior.**
   - **Numbers last.** File list, line ranges, and diff stats, if useful at all.
4. Open with the full section list as an outline. After each section, repeat it with finished sections marked and the next highlighted, so the reader always knows where they are.

## Writing rules (Simplified Technical English)

- One topic per sentence, 20 words max (25 in step-by-step instructions).
- Active voice that names the actor: "the script writes the file".
- Simple present or simple past; imperative for instructions.
- Paragraphs of 6 sentences max.
- One meaning per word, one word per thing, preferring the names the code and PR already use.
- Noun clusters of three words max; break longer ones up or add a preposition.
- Exact numbers or none; "some", "several", "significant" say nothing.
- Concrete verbs: "fetch", "merge", "post" over "handle", "process", "manage".

## Rules

- Less detail than feels natural: 5 to 8 short paragraphs or tour stops in all, each stop 2 to 4 sentences plus at most one `<pre>`.
- Explain the mechanism once, at the level of modules and seams; leave line-by-line commentary, helper functions, and whatever the diff already shows to the diff.
- Trace what the code actually does; names can lie.
- Say when the evidence falls short of a conclusion.

## `visual` mode

`/pr-explainer visual` adds a one-page visual companion to the chat explanation. The page is the map and the code; the chat is where the reader asks questions.

**Reader.** An engineer new to this system who thinks in packages, types, and functions, has read none of the code, and wants names they can grep.

**Steps.** Write the chat explanation as usual. Copy `template.html` to `_scratch/pr-explainer/<branch>.html` and fill its `FILL` slots top to bottom, following their comments and matching `example.html` in shape and density. The template's CSS and JS stay fixed and place every box and wire, so the page is built only from those slots. `open` it and end the reply with the path.

**Beyond the slot comments:**
- Split into lanes whenever the story line would need "one lane at a time".
- Box phrases use plain words, free of diff symbols and shorthand.
- One encoding: blue = this PR touched it (click → its code pair), gray = context (click → description only). The legend states it; every click obeys it.

**Before opening, confirm:** every label survives a reader who has seen no code · every kind tag and path is true · one encoding, stated, obeyed by every click · every code side has its sentence · nothing hand-placed.
