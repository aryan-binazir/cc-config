---
description: Show API changes in modified files from Git
---

Show API changes in modified files from Git:

1. Check root directory for CLAUDE.md and AGENTS.md for project-specific instructions
2. Get changed files from Git (staged, unstaged, or branch changes)
3. Analyze API changes in modified files:
   - Function signatures (parameters, return types)
   - Class/interface definitions
   - Endpoint routes and methods
   - Public method additions/removals
   - Breaking changes detection
4. Generate structured diff showing:
   - Added APIs (new functions, endpoints, classes)
   - Modified APIs (changed signatures, parameters)
   - Removed APIs (deleted or made private)
   - Breaking vs non-breaking changes
5. Export results in readable format

Supported languages:
- JavaScript/TypeScript: functions, classes, exports
- Python: functions, classes, decorators
- Go: functions, structs, interfaces
- Java: methods, classes, interfaces
- REST APIs: OpenAPI/Swagger changes

Usage: Run from project root directory