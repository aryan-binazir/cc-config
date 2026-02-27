---
description: Analyze branch changes and generate concise bullet points for JIRA ticket summaries
version: "1.1"
---

# Generate JIRA Ticket Summary

Analyze all commits on the current branch since it diverged from the base branch and create a concise bullet-point summary suitable for copying into a JIRA ticket.

## Process:

1. **Detect branch context:**
   - Current branch: `git branch --show-current`
   - Auto-detect base branch: `git symbolic-ref refs/remotes/origin/HEAD | cut -d'/' -f4`
   - Fallback order: main → master → develop
   - Validate base branch exists: `git rev-parse --verify origin/<base-branch>`

2. **Gather commit information:**
   - All commits: `git log --oneline <base-branch>..HEAD`
   - Commit count: `git rev-list --count <base-branch>..HEAD`
   - Time range: `git log --pretty=format:'%ai' <base-branch>..HEAD | head -1` and `tail -1`

3. **Analyze changes:**
   - Files changed: `git diff --name-status <base-branch>...HEAD`
   - Change summary: `git diff --stat <base-branch>...HEAD`
   - Identify file types and components affected

4. **Generate summary:**
   - Categorize by type: features, bugfixes, refactoring, tests, docs
   - Focus on business value and user impact
   - Highlight breaking changes or deployment requirements
   - Keep technical details minimal

## Output Template:

```
# Branch: [branch-name]
**Commits:** [count] | **Files Changed:** [count] | **Period:** [date-range]

## 🎯 Key Accomplishments
- [Most important business value delivered]
- [Secondary features or fixes]
- [Supporting changes or improvements]

## 📋 Technical Summary  
- **Features**: [count] new capabilities added
- **Bugfixes**: [count] issues resolved
- **Refactoring**: [count] code improvements
- **Tests**: [count] test files updated
- **Documentation**: [count] docs updated

## 🚀 Deployment Notes
- [Any breaking changes or migration requirements]
- [Database changes or configuration updates needed]
- [Performance or security improvements]

## 📁 Components Modified
- [List of main areas/modules affected]
```

## Guidelines:

- **Business first**: Lead with user-facing value and impact
- **Concise bullets**: One line per accomplishment, action-oriented
- **Group related**: Cluster similar changes together  
- **Flag risks**: Highlight breaking changes or deployment needs
- **Skip implementation**: Focus on what was delivered, not how
- **Ready to paste**: Format for direct use in JIRA ticket descriptions

Usage: Run from any branch to generate summary against detected base branch