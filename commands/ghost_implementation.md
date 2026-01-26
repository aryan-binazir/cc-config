---
name: ghost_implementation
description: Add ghost comments to code files as implementation scaffolding
argument-hint: "<plan location: file path or 'chat'>"
version: "1.0"
---

Add ghost comments to source files as scaffolding before implementation. These comments mark where changes will be made and include function stubs, data structures, and rationale.

## Plan Location

$ARGUMENTS

- If a file path: Read the plan from that file
- If `chat` or empty: Use the plan discussed/agreed upon in this conversation
- If unclear: Ask the user to clarify where the plan is

## Ghost Comment Format

Use language-appropriate comment syntax. Structure:

```
// GHOST: <function/component name>
// WHY: <rationale for this change>
// IN:  <input data structure with types>
// OUT: <output data structure with types>
// SAMPLE: <optional example with real values>
// function signature or stub here
```

**Examples by language:**

TypeScript/JavaScript:
```typescript
// GHOST: syncUserPreferences
// WHY: Enables cross-device preference sync for premium users
// IN:  { userId: string, preferences: { theme: string, notifications: boolean } }
// OUT: { success: boolean, syncedAt: Date, conflictsResolved: number }
// SAMPLE: syncUserPreferences("user_123", { theme: "dark", notifications: true })
//         => { success: true, syncedAt: 2024-01-15T10:30:00Z, conflictsResolved: 0 }
async function syncUserPreferences(userId: string, preferences: UserPreferences): Promise<SyncResult> {
  // TODO: implement
  throw new Error("Not implemented");
}
```

Python:
```python
# GHOST: sync_user_preferences
# WHY: Enables cross-device preference sync for premium users
# IN:  user_id: str, preferences: dict[str, Any]
# OUT: SyncResult { success: bool, synced_at: datetime, conflicts_resolved: int }
# SAMPLE: sync_user_preferences("user_123", {"theme": "dark", "notifications": True})
#         => SyncResult(success=True, synced_at=..., conflicts_resolved=0)
def sync_user_preferences(user_id: str, preferences: dict[str, Any]) -> SyncResult:
    # TODO: implement
    raise NotImplementedError()
```

Go:
```go
// GHOST: SyncUserPreferences
// WHY: Enables cross-device preference sync for premium users
// IN:  userID string, preferences map[string]interface{}
// OUT: (*SyncResult, error)
// SAMPLE: SyncUserPreferences("user_123", map[string]interface{}{"theme": "dark"})
//         => &SyncResult{Success: true, SyncedAt: time.Now()}, nil
func SyncUserPreferences(userID string, preferences map[string]interface{}) (*SyncResult, error) {
	// TODO: implement
	return nil, errors.New("not implemented")
}
```

## Workflow

### Step 1: Parse the Plan

Read and understand the implementation plan. Identify:
- What functions/components need to be created or modified
- Where in the codebase each change belongs
- The data flow (inputs/outputs) for each piece
- The rationale for each change

### Step 2: Locate Target Files

For each planned change:
- Identify the correct file (existing or new)
- Find the appropriate location within the file (near related code)
- If file doesn't exist, note it will be created

### Step 3: Write Ghost Comments

For each change location:
1. Add the GHOST comment block with:
   - Function/component name
   - WHY (rationale from the plan)
   - IN (input data structure with types)
   - OUT (output data structure with types)
   - SAMPLE (optional, include if it aids clarity)
2. Add a minimal function stub that throws/raises "not implemented"
3. Use the file's existing code style (indentation, naming conventions)

### Step 4: Report

After writing all ghost comments, output a summary:

```
## Ghost Implementation Scaffolding

Added ghost comments to X files:

- `path/to/file.ts:45` - functionName (WHY: brief reason)
- `path/to/other.ts:120` - otherFunction (WHY: brief reason)
- ...

Run `/clean_ghost` to remove all ghost comments when implementation is complete.
```

## Rules

- Match the language's comment syntax and conventions
- Place ghosts near related existing code when possible
- Keep stubs minimal - just signature + throw/raise
- Don't modify existing functional code
- If a file needs to be created, create it with just the ghost comment
- SAMPLE is optional - include when the data structure is complex or non-obvious
