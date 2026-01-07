---
description: Generate reviewable patch with summary for current changes
version: "1.0"
---

# Patch Summary Command

Generate a clean git diff patch with AI-generated summary for review.

## Process:

1. **Get current changes:**
   - Check `git status -sb` for branch info
   - Generate unified diff: `git diff HEAD` (or `git diff --cached` if only staged)
   - Get changed files list: `git diff --name-status HEAD`
   - Count insertions/deletions: `git diff --stat HEAD`

2. **Analyze changes:**
   - Read the full diff output
   - Identify the nature of changes (feature, fix, refactor, etc.)
   - Note affected files and components
   - Check CONTEXT.md for current objective alignment

3. **Generate patch file:**
   - Create `patch-YYYY-MM-DD-HHMMSS.patch` with full diff
   - Add summary header to patch file:
     ```
     # Summary
     [2-3 bullet points of what changed and why]

     # Files Changed
     [List with change counts]

     # Review Notes
     [Any blockers, risks, or questions for reviewer]
     ```

4. **Output:**
   - Display patch summary to console
   - Show patch file location
   - Provide git apply instructions for reviewer

## Guidelines:
- Focus summary on "why" not "what" (code shows "what")
- Flag any breaking changes or risky modifications
- Note if changes diverge from CONTEXT.md objectives
- Keep summary under 10 lines

Usage: Run from project root. Works with staged or unstaged changes.
