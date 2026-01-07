---
name: code-review
description: Systematic code review using checklists. No compliments, no padding.
version: "2.0"
category: code-quality
---

# Code Review Skill

Systematic review using checklists. List issues only. No compliments. No padding.

## Review Checklist

### Security
- **Auth**: Authentication/authorization at right layer? Privilege escalation risks?
- **Input**: Validated server-side (length, type, format)? SQL injection, XSS, command injection?
- **Data**: Hardcoded secrets? Sensitive data logged? PII handled correctly? Passwords hashed?
- **Common vulns**: Path traversal, SSRF, XXE, deserialization, race conditions, CSRF, open redirects
- **Dependencies**: Up to date? Known vulnerabilities? Security headers configured?

### Performance
- **Algorithm**: Time complexity appropriate? Unnecessary iterations or repeated work?
- **Data access**: N+1 queries? Missing indexes? Fetching more than needed? Can queries batch?
- **Memory**: Large objects loaded unnecessarily? Leaks? Unbounded caches? Streaming available?
- **I/O**: Blocking critical paths? Parallelizable? Timeouts set? Resources closed?
- **Concurrency**: Race conditions? Lock contention? Deadlocks?

### Error Handling
- **Exceptions**: Caught at right level? Context preserved? Logged with detail? Too broad?
- **Edge cases**: Null/empty/zero/negative/boundary values? Network/DB failures? Large inputs?
- **Resources**: Cleaned up in all paths? try-finally/defer/using correct? Cleanup on failure?

### Readability
- **Naming**: Descriptive, consistent, reveals intent? Boolean names clear (is/has/can)?
- **Functions**: Single responsibility? Too long (>50 lines)? Too many params (>4)?
- **Organization**: Logical grouping? Separation of concerns? File too large?
- **Complexity**: Nesting >3 levels? Convoluted conditionals? Extract functions?

### Maintainability
- **Smells**: Duplication? God objects? Feature envy? Primitive obsession?
- **Testing**: Testable? Tests included/updated? Edge cases covered?
- **Dependencies**: Necessary? Tight coupling? Circular dependencies?

### Language-Specific
- **Python**: List comprehensions? Context managers? Type hints? `with` for files?
- **JS/TS**: Proper async/await? Promise error handling? Avoiding `any`?
- **Java**: Try-with-resources? Streams? StringBuilder over `+`?
- **Go**: Errors not ignored? Defer for cleanup? Context propagation?
- **Rust**: Ownership/borrowing correct? Result for errors? Avoiding unnecessary clones?

### Database/API
- **Schema**: Indexes appropriate? Foreign keys? Constraints?
- **Queries**: Efficient? Paginated? Transactions correct?
- **API**: Consistent naming? Versioned? Breaking changes avoided?
- **Request/Response**: Validated? Error responses consistent? Rate limits?

## Output

List only issues that need fixing.

```
## Issues

1. [file:line] - [what's wrong and why it matters]
2. [file:line] - [what's wrong and why it matters]
...

## Concerns

- [Any security, performance, or architectural concerns]

## Verdict
[APPROVE / NEEDS FIXES / REJECT] - [1 sentence summary]
```

If no issues found, say so and move on.

## Red Flags

```python
# SQL injection
query = f"SELECT * FROM users WHERE name = '{user_input}'"  # BAD
cursor.execute("SELECT * FROM users WHERE name = %s", (user_input,))  # GOOD

# N+1 queries
for user in users:
    orders = db.query(f"SELECT * FROM orders WHERE user_id = {user.id}")  # BAD
# Use JOIN instead

# Silent failure
except: pass  # BAD
except SpecificException as e: logger.error(f"Failed: {e}")  # GOOD
```

## Escalate When

- Major architectural changes
- Security concerns you're uncertain about
- Performance implications at scale
- Breaking API changes
