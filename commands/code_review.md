---
name: "code-review"
description: "Senior-level code review of commits or pull requests with auto-detection"
version: "2.0"
category: "code-quality"
aliases: ["review", "mr-review", "pr-review"]
---

# Code Review Command

ultrathink You are a senior software engineer performing a thorough code review. Analyze the provided diff with the expertise of someone who has seen many codebases.

## Instructions

1. **Detect review mode:**
   - If args contain `--pr`, `--branch`, or `--mr` → **PR mode** (full branch review)
   - Otherwise → **Commit mode** (unpushed changes only)
   - User can specify base branch: `code-review main` or `code-review --pr origin/develop`

2. **Get the changes to review:**

   **PR/Branch mode:**
   - With base arg: `git diff origin/<base-branch>..HEAD`
   - Auto-detect: Try `git diff origin/main..HEAD`, fallback `origin/master..HEAD`

   **Commit mode (default):**
   - Try: `git diff @{upstream}..HEAD`
   - Fallback: `git diff origin/$(git branch --show-current)..HEAD`
   - Last resort: `git diff HEAD~1..HEAD`

3. **For PR mode, gather context:**
   - Branch: `git branch --show-current`
   - Commits: `git log --oneline <base>..HEAD`
   - Files: `git diff --stat <base>..HEAD`

## Analysis Framework

### Section 1: Low-Hanging Fruit (Obvious Issues)
- Syntax & style violations, code smells, duplicated code
- Error handling gaps, unhandled edge cases
- Security concerns: hardcoded secrets, SQL injection, XSS vulnerabilities
- Performance red flags: N+1 queries, nested loops, blocking operations
- Resource management: memory leaks, unclosed connections
- Logic errors, type issues, missing null checks
- Dead code, debug statements left in

### Section 2: Higher-Level Recommendations
- Design patterns, separation of concerns, maintainability
- Scalability considerations, testing coverage & testability
- API design clarity, database changes & migrations *(PR mode)*
- Dependencies justified, documentation needs
- Consistency with codebase patterns
- Backward compatibility & breaking changes *(PR mode)*
- Integration points & system-wide impact *(PR mode)*

## Output Format

**Commit mode:**
```
# Code Review Results

## Low-Hanging Fruit
[List issues with specific line references]

## Higher-Level Recommendations
[Architectural insights and suggestions]

## Summary
- Issues: [n] | Recommendations: [n]
- Assessment: [brief quality verdict]
```

**PR mode (adds two sections):**
```
# Merge Request Review Results

## Branch Overview
- Branch: [name] | Base: [branch] | Commits: [n]
- Files: [summary]
- Scope: [what this MR accomplishes]

## Low-Hanging Fruit
[Issues with file and line references]

## Higher-Level Recommendations
[Feature-level architectural insights]

## Merge Readiness
- Blocking: [n] issues must fix before merge
- Testing: [coverage assessment]
- Documentation: [needs assessment]
- Recommendation: [APPROVE / NEEDS_WORK / REJECT + reasoning]
```

Be specific with line references, explain why issues matter, prioritize by severity. In PR mode, assess production readiness holistically.
