---
name: pr-explainer
description: Explain the current pull request in concise, plain language. Use when the user wants to understand what a PR changes, why it exists, and how it fits into the larger system.
---

# PR Explainer

Explain the current pull request to someone unfamiliar with the change. Always place the ticket within the larger Epic or set of changes being implemented — spell that context out for the user.

## Workflow

1. Read the PR, its ticket or stated goal, and the diff.
2. Inspect enough surrounding code to understand where the change sits in the system; the diff alone is too thin a basis.
3. Explain the change in this order:
   - The larger system goal and the affected concept's place within it.
   - The ticket's goal and how it advances that larger goal.
   - What the PR changes and how the important pieces work together.
   - Relevant verification, limitations, or intentionally unchanged behavior.

## Rules

- Keep the explanation concise, with only enough system context to orient the reader.
- Use plain language and concrete terms.
- Trace what the code actually does rather than trusting names.
- Group changes by purpose instead of listing every changed file.
- Say when the available evidence falls short of a conclusion.
