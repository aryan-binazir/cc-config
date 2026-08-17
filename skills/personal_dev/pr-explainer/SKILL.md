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

## `html` mode

When invoked with `html` (e.g. `/pr-explainer html`), produce a one-page visual companion instead of prose. Prose mode is the full explanation; html mode is the map and the code.

### Reader

An engineer who does not know this system and needs to understand it. They think in packages, types, and functions. Write for a reader who has seen none of the code, and use the names the code uses.

### Files

- `template.html` in this skill's directory: the page with all CSS and JS in place. Fill the `FILL` slots only; leave everything below `do not edit` alone.
- `example.html` in this skill's directory: the template filled in for a fictional PR. Match its shape and density.

### Steps

1. Copy `template.html` to `_scratch/pr-explainer/<branch>.html`.
2. Fill the slots, top to bottom, following the rules below.
3. Run `open <file>` and reply with the path only.

### The page, top to bottom

**Title.** PR number and one plain line saying what it does.

**System / Problem / Fix.** One sentence each, before anything visual. System: what this service is and the path a request or piece of data takes through it, naming the packages. Problem: what was wrong or missing, concretely. Fix: what this PR does, naming `pkg.Symbol`.

**Story line.** One sentence the reader can say aloud while following the arrows left to right, using the real symbols.

**Map.** The change path, not the whole system: the packages a changed call passes through, about eight boxes at most. Left to right in the direction data flows.
- Every box sits inside a container labelled `package <name>`, so the reader sees which symbols share a module.
- Box title is the code's own symbol, `pkg.Name`, in monospace. Above it a kind tag (func, type, method) that is true. Below it one plain phrase saying what it does now, e.g. "no longer checks the limit itself", "new type: counts requests per tenant". For untouched boxes the phrase is "unchanged".
- Blue box = this PR touched it; gray box = shown for context. The legend states this and every interaction obeys it: clicking a blue box highlights its code pair; clicking a gray box shows its description only.
- Each box carries `data-desc` (what it is, one sentence) and `data-role` (what this PR does to it, one sentence). Untouched boxes get both too.
- Arrows mean "calls". Label a wire only when the label adds information the two boxes lack: a method or interface name that is not the box title, or a seam this PR added or changed.
- Other callers on the path go in the "also" line as text ("4 other packages call `orchestrator.Run`; unchanged"), not as boxes.
- Boxes size to their text and wires are drawn by the script from real positions. Add boxes to `.col`/`.group` elements and wires to `W`; write no coordinates.

**Code pairs.** One to three before/after pairs, each tied to a blue box.
- Title is the symbol that changed, plus a monospace line naming the file or files. When before and after live in different files, say "removed from … · added to …".
- Each side opens with one plain sentence saying what that code does or meant, then the snippet. Bold the one word whose meaning changed (e.g. connection → tenant).
- Removed lines carry `class=d`, added lines `class=a`. Keep lines short enough that both columns fit side by side.

**Footer.** One line for what was verified and one for what was deliberately left unchanged or is out of scope. Delete it if there is nothing to say.

### Checks before opening

- Every label survives a reader who has seen no code: real symbols, plain phrases, no diff shorthand or symbols in box text.
- Every kind tag and path is true for the symbol it sits on.
- One visual encoding, stated in the legend, obeyed by every click.
- Every code side has its sentence.
- Nothing hand-placed: no coordinates, no elements added outside the marked slots.
