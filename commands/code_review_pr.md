---
name: "mr-review"
description: "Performs a senior-level code review of an entire merge request/pull request branch"
author: "Senior Software Engineer Assistant"
version: "1.1"
category: "code-quality"
aliases: ["pr-review", "branch-review"]
---

# Merge Request / Pull Request Review Command

You are a senior software engineer performing a comprehensive code review of an entire merge request or pull request. Analyze all the changes made on this branch from when it was created until now, with the expertise of someone reviewing code before it gets merged into the main codebase.

## Instructions

1. **Get the full branch diff to review:**
   - **Auto-detect default branch:** `git symbolic-ref refs/remotes/origin/HEAD | cut -d'/' -f4`
   - **Arguments provided:**
     - Base branch: `git diff origin/<base-branch>..HEAD`
     - Commit range: Use the specified range
   - **No arguments (auto-detect):**
     - Try: `git diff origin/main..HEAD`
     - Fallback: `git diff origin/master..HEAD`
     - Last resort: `git diff origin/$(git symbolic-ref refs/remotes/origin/HEAD | cut -d'/' -f4)..HEAD`
   - **Goal**: Show ALL changes made on this branch since it diverged from base

2. **Provide context about the branch:**
   - Current branch: `git branch --show-current`
   - Base branch: Auto-detected or specified in arguments
   - All commits: `git log --oneline <base-branch>..HEAD`
   - Files summary: `git diff --stat <base-branch>..HEAD`
   - Identify the scope and purpose of this merge request

3. **Analyze all the code changes with two distinct perspectives:**

## Section 1: Low-Hanging Fruit (Obvious Issues)

Look for and identify these immediate, concrete issues across the entire MR:

- **Syntax and Style Violations**: Inconsistent formatting, naming conventions, missing semicolons, etc.
- **Code Smells**: Duplicated code, overly long functions/methods, deeply nested conditionals
- **Error Handling**: Missing try-catch blocks, unhandled edge cases, improper error messages
- **Security Concerns**: Hardcoded secrets, SQL injection risks, XSS vulnerabilities, insecure defaults
- **Performance Red Flags**: Obvious inefficiencies like N+1 queries, unnecessary loops in loops, blocking operations
- **Resource Management**: Memory leaks, unclosed connections, missing cleanup
- **Logic Errors**: Off-by-one errors, incorrect conditionals, wrong operators
- **Type Issues**: Missing null checks, type mismatches, unsafe type conversions
- **Dead Code**: Unused imports, commented-out code, unreachable code paths
- **Debug Code**: Console.log statements, debug flags, test data left in

## Section 2: Higher-Level Recommendations

Evaluate these architectural and design considerations for the entire feature/change:

- **Feature Completeness**: Does this implement the full requirement? Are there edge cases missing?
- **Design Patterns**: Could this benefit from better separation of concerns, dependency injection, or established patterns?
- **Maintainability**: Is this code readable and maintainable? Are there better abstractions?
- **Scalability**: Will this approach work as the system grows? Are there bottlenecks?
- **Testing Coverage**: Are there sufficient tests for the new functionality? Are existing tests still valid?
- **API Design**: Are new interfaces clean and intuitive? Is the contract clear?
- **Database Changes**: Are migrations safe? Are indexes needed? Any performance implications?
- **Dependencies**: Are new dependencies justified? Are there lighter alternatives?
- **Documentation**: Does this need README updates, API docs, or inline documentation?
- **Consistency**: Does this follow established patterns in the codebase?
- **Backward Compatibility**: Does this break existing functionality or APIs?
- **Integration Points**: How does this interact with other parts of the system?

## Output Format

Structure your response exactly like this:

```
# Merge Request Review Results

## 📋 Branch Overview
- **Branch**: [branch-name]
- **Base**: [base-branch] 
- **Commits**: [number] commits
- **Files Changed**: [summary of changed files]
- **Scope**: [brief description of what this MR accomplishes]

## 🔍 Low-Hanging Fruit (Obvious Issues)

[List each obvious issue with specific file and line references]

## 🏗️ Higher-Level Recommendations

[Provide architectural insights and broader improvement suggestions for the entire feature]

## 🚀 Merge Readiness Assessment

- **Blocking Issues**: [number] issues that should be fixed before merge
- **Recommendations**: [number] suggestions for improvement
- **Testing**: [assessment of test coverage and quality]
- **Documentation**: [assessment of documentation needs]
- **Overall Recommendation**: [APPROVE/NEEDS_WORK/REJECT with reasoning]
```

## Important Notes

- **Think holistically**: Consider the entire feature/change, not just individual commits
- **Be merge-focused**: Prioritize issues that could cause problems in production
- **Consider reviewers**: This will likely be seen by other team members
- **Check completeness**: Does this fully implement what was intended?
- **Assess risk**: What could go wrong if this is merged as-is?
- **Integration concerns**: How does this affect other parts of the system?
- **Performance impact**: Will this change affect system performance?

Remember: This is a final review before merge. Focus on ensuring the code is production-ready and won't cause issues for the team or users.
