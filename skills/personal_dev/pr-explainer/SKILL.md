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

- Keep the explanation concise, with only enough system context to orient the reader.
- Trace what the code actually does rather than trusting names.
- Group changes by purpose instead of listing every changed file.
- Say when the available evidence falls short of a conclusion.
