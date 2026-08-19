---
name: pr-explainer
description: Explain the current pull request in concise, plain language. Use when the user wants to understand what a PR changes, why it exists, and how it fits into the larger system.
---

# PR Explainer

Explain the current pull request to a reader who is starting cold. Orient them in the system first; show code and numbers last.

## Workflow

1. Read the PR, its ticket or stated goal, and the diff.
2. Inspect enough surrounding code to understand where the change sits in the system; the diff alone is too thin a basis.
3. Explain the change in this order:
   - **System context.** The larger goal (epic, feature set, or workflow) and the concept this PR touches. No file names, line counts, or diff stats here.
   - **The ticket's goal** and how it advances that larger goal.
   - **What the PR changes** and how the important pieces work together, grouped by purpose.
   - **Verification, limitations, and intentionally unchanged behavior.**
   - **Numbers last.** File list, line ranges, and diff stats, if useful at all.
4. Open with the full section list above as an outline. After each section, repeat the outline with the finished sections marked and the next one highlighted, so the reader always knows where they are.

## Writing rules (Simplified Technical English)

- One topic per sentence. Sentences of 20 words or fewer; 25 in step-by-step instructions.
- Active voice. Name the actor: "the script writes the file", not "the file is written".
- Simple present or simple past. Imperative for instructions.
- Paragraphs of 6 sentences or fewer.
- One meaning per word; reuse the same term for the same thing throughout. Prefer the names the code and PR already use.
- No noun clusters longer than three words. Break them up or add a preposition.
- No vague quantifiers ("some", "several", "significant"). Give the number or omit it.
- Concrete verbs over abstractions: "fetch", "merge", "post", not "handle", "process", "manage".

## Rules

- Less detail than feels natural. Whole explanation: 5 to 8 short paragraphs, or 5 to 8 tour stops. Each stop: 2 to 4 sentences plus at most one `<pre>`.
- Explain the mechanism once at the level of modules and seams; skip line-by-line commentary, helper functions, and anything the reader can see in the diff.
- Keep the explanation concise, with only enough system context to orient the reader.
- Trace what the code actually does rather than trusting names.
- Group changes by purpose instead of listing every changed file.
- Say when the available evidence falls short of a conclusion.
- After pushing a fix based on a PR comment, reply directly to that comment with the commit hash, alone or in a sentence.

## `visual` mode

`/pr-explainer visual` gives the normal prose explanation in chat **and** a one-page visual companion. The page is the map and the code; the chat is where the reader asks questions.

**Reader.** An engineer who does not know this system. Thinks in packages, types, functions. Has read none of the code. Wants names they can grep.

**Files.** Always use `template.html`; do not create your own HTML. Its CSS/JS is fixed, so fill the `FILL` slots only. `example.html`: the template filled for a fictional PR; match its shape and density.

**Steps.** Write the prose explanation in chat as usual. Copy `template.html` to `_scratch/pr-explainer/<branch>.html`, fill it top to bottom, `open` it, and end the reply with the path.

**Page, top to bottom.**
- Title: PR number + one plain line.
- System / Problem / Fix, one sentence each, before any picture. System names the packages a request passes through. Fix names `pkg.Symbol`.
- Lanes: one `.lane` per independent change in the PR. One change → one lane, no heading. Several → each lane gets a short heading, its own story line, its own map, its own "also" line. If the story line needs "one lane at a time", split into lanes.
- Story line: one sentence readable aloud while following the arrows, using real symbols.
- Map: one change path per lane, not the system. ~8 boxes, left to right in the direction data flows. Each package appears once per lane.
  - Every box lives in a container labelled `package <name>`.
  - Box = kind tag (func / type / method, true) + `pkg.Symbol` + one plain phrase saying what it does now ("no longer checks the limit itself"; "unchanged" for context boxes). Plain words only; no diff symbols or shorthand.
  - Blue = this PR touched it; gray = context. The legend says so; every click obeys it (blue → its code pair, gray → description only).
  - Every box has `data-desc` (what it is) and `data-role` (what this PR does to it), one sentence each. Context boxes too.
  - Arrows mean "calls". Label a wire only when the boxes alone don't tell you: a method or interface not in the box title, or a seam this PR added.
  - Other callers go in the "also" line as text, not as boxes.
  - Add boxes to `.col` / `.group`, wires to `W`. The script sizes boxes, draws wires, and fixes the info panel's height. Write no coordinates, add no elements outside the slots.
- Code pairs: one to three, each tied to a blue box.
  - Title = the symbol that changed + a mono line naming the files. Different files before and after: "removed from … · added to …".
  - Each side: one plain sentence saying what that code does or meant, then the snippet. Bold the one word whose meaning changed.
  - `class=d` removed lines, `class=a` added lines. Short lines so both columns fit.
- Footer: one line verified, one line deliberately unchanged / out of scope. Delete if empty.

**Before opening, confirm:** every label survives a reader who has seen no code · every kind tag and path is true · one encoding, stated, obeyed by every click · every code side has its sentence · nothing hand-placed.
