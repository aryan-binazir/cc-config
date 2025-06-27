---
description: Analyze branch changes and generate concise bullet points for JIRA ticket summaries
allowed-tools: Bash(git*)
---

# Generate JIRA Ticket Summary

Analyze all commits on the current branch since it diverged from main and create a concise bullet-point summary suitable for copying into a JIRA ticket.

## Analysis Steps:

1. **Get branch context:**
   - Current branch name: !`git branch --show-current`
   - Commits since main: !`git log --oneline main..HEAD`

2. **Analyze changes:**
   - Files changed: !`git diff --name-status main...HEAD`
   - Full diff summary: !`git diff --stat main...HEAD`

3. **Generate summary:**
   - Categorize changes by type (features, bugfixes, refactoring, etc.)
   - Create concise bullet points focusing on what was accomplished
   - Highlight any breaking changes or important considerations
   - Keep technical details minimal - focus on business value

## Output Format:

Generate a clean markdown summary with:
- **Summary** section with bullet points of key changes
- **Files Modified** section showing count and main areas
- **Branch Info** showing branch name and commit count

## Guidelines:

- Each bullet point should be concise and actionable
- Lead with the most impactful changes
- Group related changes together
- Focus on what was accomplished, not how it was implemented
- Use business-friendly language suitable for JIRA tickets
- Highlight any deployment considerations or dependencies

The output should be ready to copy and paste directly into a JIRA ticket description.