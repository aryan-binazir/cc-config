---
description: Scan current work for potential blockers and escalation points
version: "1.0"
---

# Blocker Scan Command

Proactively identify blockers, risks, and items requiring escalation.

## Process:

1. **Check context alignment:**
   - Read CONTEXT.md (or branch-specific context file)
   - Compare current changes with stated objectives
   - Flag if work diverges from plan
   - Identify any unclear requirements

2. **Analyze current changes:**
   - Run `git diff HEAD` to see all changes
   - Scan for TODO, FIXME, HACK, XXX comments in changed code
   - Look for commented-out code blocks
   - Check for hardcoded values that need configuration

3. **Dependency and breaking change check:**
   - Identify modified public APIs/interfaces
   - Check for removed functions/classes
   - Look for changed function signatures
   - Flag potential backward-incompatible changes

4. **External dependency scan:**
   - Check for new imports/dependencies in changed files
   - Verify new dependencies are declared in package.json/requirements.txt/go.mod
   - Flag missing dependency declarations

5. **Report blockers:**

## Output Format:
```
BLOCKER SCAN RESULTS
====================
Status: [RED|YELLOW|GREEN]

RED - ESCALATE NOW:
- [Blocker description]

YELLOW - FLAG FOR REVIEW:
- [Issue description]

GREEN - ALL CLEAR:
- Context aligned
- No breaking changes
```

## Severity Levels:
- **RED**: Breaking changes, missing requirements, unclear objectives
- **YELLOW**: TODOs, hardcoded values, new dependencies
- **GREEN**: No blockers detected

Usage: Run before creating patches or PRs to catch issues early.
