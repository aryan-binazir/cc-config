---
name: mermaid
description: Generate Mermaid diagrams for system understanding
version: "2.0"
argument-hint: [--md] [--branch | --system <target> | <description>]
---

# Mermaid Diagram Generator

Generate clear, verifiable Mermaid diagrams that balance high-level understanding with enough detail to confirm correctness.

## User Request

$ARGUMENTS

If `$ARGUMENTS` is empty, ask: "What should I diagram? Options: `--branch` (changes on this branch), `--system <target>` (specific system/flow), or describe what you want to visualize."

## Modes

- `--branch`: Analyze changes on current branch vs main. Produce a diff-aware data flow diagram showing how data moves through the changed code, with clear separation of changed vs unchanged context.
- `--system <target>`: Analyze a specific system, module, or flow. `<target>` can be a file path, directory, function name, or concept (e.g., "auth flow", "payment processing").
- **Freeform**: Any other input is treated as a description of what to diagram.

## Output Options

- **Default**: HTML file with rendered diagram (self-contained, opens in browser)
- `--md`: Output both HTML and markdown files

## Workflow

### 1. Gather Context

**For `--branch`:**
```bash
BASE=$(git merge-base origin/main HEAD 2>/dev/null || git merge-base origin/master HEAD)
git diff --stat $BASE..HEAD
git diff $BASE..HEAD
```

**For `--system <target>`:**
- Locate relevant files using grep/glob
- Read entry points, core logic, and data models
- Trace the flow through the system

**For freeform:**
- Parse the description to understand what needs diagramming
- Gather relevant code context

### 2. Analyze & Design

Identify:
- **Entry points**: Where data/control enters
- **Transformations**: What happens to data at each step
- **Decision points**: Branches, conditions, error paths
- **Exit points**: Where data/control leaves (returns, side effects, external calls)
- **Key actors**: Services, modules, functions, external systems

**Additional analysis for `--branch`:**
- **Changed nodes**: Functions, modules, or components that were added, modified, moved, or deleted
- **Primary entry points touched**: Entry points whose behavior changed in this branch
- **External touchpoints affected**: DB tables, external services, APIs, emitted events impacted by the diff
- **Context nodes**: Minimal set of unchanged nodes required to make the changed flow understandable

### 3. Select Diagram Type

Choose the most appropriate type:

| Type | Use When |
|------|----------|
| `flowchart TD` | Control flow, decision trees, process steps |
| `flowchart LR` | Data pipelines, request/response flows |
| `sequenceDiagram` | Multi-actor interactions, API calls, async flows |
| `stateDiagram-v2` | State machines, lifecycle, status transitions |
| `classDiagram` | Data models, relationships, inheritance |
| `erDiagram` | Database schemas, entity relationships |

Default to `flowchart LR` for data flow unless another type is clearly better.

### 4. Diagram Style Rules

**Clarity over completeness:**
- Max 15-20 nodes per diagram. Split into multiple diagrams if needed.
- Use descriptive but concise labels (verb + noun: "Validate Input", "Fetch User")
- Group related nodes with subgraphs when it aids understanding

**Detail level:**
- Include function/file names where they help verification
- Show data shape at key boundaries (e.g., `{userId, token}`)
- Mark external systems distinctly (use `[(Database)]` or `{{External API}}`)

**Visual conventions:**
```
[Rectangle] - Process/function
([Stadium]) - Start/end points
{Diamond} - Decision
[(Cylinder)] - Database/storage
{{Hexagon}} - External service
[[Subroutine]] - Reusable component
```

**Edge conventions:**
- Label edges with data being passed or condition
- Use dotted lines `-.->` for optional/async/stream/event paths
- Use thick lines `==>` for primary/happy path

### 5. Branch-Specific Diagram Rules (`--branch` only)

These rules apply on top of the base style rules when operating in `--branch` mode.

#### 5a. Diff-Aware Node Labels

Append a status tag to every node that was part of the diff:

| Tag | Meaning |
|-----|---------|
| `(new)` | Newly introduced node/function |
| `(modified)` | Changed node/function |
| `(moved)` | Code relocated to a different file/module |
| `(deleted)` | Removed flow (include only when the deletion affects the visible path) |

Unchanged context nodes get no tag. Example labels: `Auth Middleware (modified)`, `Refresh Flow (new)`.

#### 5b. Touched vs Context Grouping

Separate changed and unchanged nodes into explicit subgraphs:

- `subgraph "Changed (this branch)"` — all nodes that were added, modified, or moved
- `subgraph "Context (unchanged)"` — minimal surrounding nodes needed to make the flow understandable

Prioritize the changed flow. Include the fewest unchanged nodes necessary for comprehension. The 15-20 node cap still applies; split into multiple diagrams if needed.

#### 5c. Edge Semantics (strict)

Apply these edge types consistently in `--branch` diagrams:

| Edge | Meaning |
|------|---------|
| `==>` | Primary changed path — the flow reviewers should follow |
| `-->` | Synchronous calls providing context (unchanged paths, supporting calls) |
| `-.->` | Async, optional, stream, or event-driven edges |

Add edge labels at system boundaries where reviewers care most: API/input boundary, DB boundary, external API boundary, emitted events.

#### 5d. Optional Impact Map

When the diff spans multiple domains or architectural layers, generate a second diagram (≤ 10 nodes) showing:

```
Changed files/packages → Impacted modules → External systems
```

Only include this when it adds clarity beyond the main diagram. Skip it for single-domain changes.

#### 5e. Scope Statement

Explicitly state what the diagram covers:
- **Included**: Flows that traverse changed code, plus minimal context
- **Excluded**: Untouched subsystems, unrelated files in the diff (if any), internal implementation detail not required for verification

### 6. Output

**Output location**: `context/diagrams/<name>.<ext>`

Where `<name>` is derived from the mode:
- `--branch`: `branch-<branch-name>-dataflow`
- `--system`: `system-<target-slug>`
- Freeform: `diagram-<slug>`

#### Default: HTML Output (`.html`)

Generate a self-contained HTML file that renders the Mermaid diagram in browser.

**For `--system` and freeform modes**, use this template:

```html
<!DOCTYPE html>
<html>
<head>
  <title>DIAGRAM_TITLE</title>
  <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 1200px; margin: 0 auto; padding: 2rem; }
    h1 { border-bottom: 1px solid #ccc; padding-bottom: 0.5rem; }
    .mermaid { margin: 2rem 0; }
    .notes { background: #f5f5f5; padding: 1rem; border-radius: 4px; margin-top: 2rem; }
  </style>
</head>
<body>
  <h1>DIAGRAM_TITLE</h1>
  <p><strong>Scope:</strong> SCOPE_DESCRIPTION</p>

  <div class="mermaid">
    MERMAID_CODE_HERE
  </div>

  <div class="notes">
    <h3>Notes</h3>
    <ul>
      <li>KEY_NOTES_HERE</li>
    </ul>
  </div>

  <script>mermaid.initialize({ startOnLoad: true });</script>
</body>
</html>
```

**For `--branch` mode**, use this extended template:

```html
<!DOCTYPE html>
<html>
<head>
  <title>DIAGRAM_TITLE</title>
  <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 1200px; margin: 0 auto; padding: 2rem; }
    h1 { border-bottom: 1px solid #ccc; padding-bottom: 0.5rem; }
    h2 { margin-top: 2rem; }
    .mermaid { margin: 2rem 0; }
    .summary { background: #f0f4ff; padding: 1rem; border-radius: 4px; margin: 1rem 0; }
    .summary table { border-collapse: collapse; width: 100%; }
    .summary td, .summary th { text-align: left; padding: 0.25rem 0.5rem; }
    .scope { background: #f5f5f5; padding: 1rem; border-radius: 4px; margin: 1rem 0; }
    .notes { background: #f5f5f5; padding: 1rem; border-radius: 4px; margin-top: 2rem; }
  </style>
</head>
<body>
  <h1>DIAGRAM_TITLE</h1>

  <div class="scope">
    <strong>Branch:</strong> BRANCH_NAME<br>
    <strong>Base ref:</strong> <code>BASE_SHA</code> (merge-base of origin/main)<br>
    <strong>Included:</strong> INCLUDED_DESCRIPTION<br>
    <strong>Excluded:</strong> EXCLUDED_DESCRIPTION
  </div>

  <h2>Change Summary</h2>
  <div class="summary">
    <table>
      <tr><th>File</th><th>Changes</th></tr>
      <!-- FILE_ROWS: <tr><td>path/to/file</td><td>+NN / −NN</td></tr> -->
    </table>
    <p><strong>Primary entry points touched:</strong> ENTRY_POINTS</p>
    <p><strong>External touchpoints:</strong> EXTERNAL_TOUCHPOINTS</p>
  </div>

  <h2>Changed Flow</h2>
  <div class="mermaid">
    MAIN_MERMAID_CODE_HERE
  </div>

  <!-- Optional: include only when diff spans multiple domains/layers -->
  <!--
  <h2>Impact Map</h2>
  <div class="mermaid">
    IMPACT_MAP_MERMAID_CODE_HERE
  </div>
  -->

  <div class="notes">
    <h3>Notes</h3>
    <ul>
      <li>KEY_NOTES_HERE</li>
    </ul>
  </div>

  <script>mermaid.initialize({ startOnLoad: true });</script>
</body>
</html>
```

#### With `--md`: Additional Markdown Output (`.md`)

When `--md` is specified, also generate a markdown file.

**For `--system` and freeform modes:**

1. **Title**: What this diagram shows
2. **Scope**: What's included/excluded
3. **Diagram(s)**: The Mermaid code block(s)
4. **Legend** (if needed): Explain non-obvious symbols
5. **Notes**: Key assumptions, simplifications, or areas needing attention

**For `--branch` mode**, use this structure:

1. **Title**: `Data Flow: <feature> (Branch Changes)`
2. **Scope**: Base ref, branch, what's included/excluded
3. **Change Summary**:
   - Files changed (+/−)
   - Primary entry points touched
   - DB tables / external systems touched
4. **Diagram 1**: "Changed Flow (with minimal context)"
5. **Diagram 2** (optional): "Impact Map" — only when diff spans multiple domains
6. **Notes**: Assumptions + verification hints (what to check in code)

## Example Output Structure

### HTML (default, `--branch` mode)

`context/diagrams/branch-oauth-refresh-dataflow.html`:

```html
<!DOCTYPE html>
<html>
<head>
  <title>Data Flow: OAuth Refresh (Branch Changes)</title>
  <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 1200px; margin: 0 auto; padding: 2rem; }
    h1 { border-bottom: 1px solid #ccc; padding-bottom: 0.5rem; }
    h2 { margin-top: 2rem; }
    .mermaid { margin: 2rem 0; }
    .summary { background: #f0f4ff; padding: 1rem; border-radius: 4px; margin: 1rem 0; }
    .summary table { border-collapse: collapse; width: 100%; }
    .summary td, .summary th { text-align: left; padding: 0.25rem 0.5rem; }
    .scope { background: #f5f5f5; padding: 1rem; border-radius: 4px; margin: 1rem 0; }
    .notes { background: #f5f5f5; padding: 1rem; border-radius: 4px; margin-top: 2rem; }
  </style>
</head>
<body>
  <h1>Data Flow: OAuth Refresh (Branch Changes)</h1>

  <div class="scope">
    <strong>Branch:</strong> feature/oauth-refresh<br>
    <strong>Base ref:</strong> <code>a1b2c3d</code> (merge-base of origin/main)<br>
    <strong>Included:</strong> Auth middleware → token validation → refresh flow<br>
    <strong>Excluded:</strong> Route handlers, unrelated middleware, session cleanup cron
  </div>

  <h2>Change Summary</h2>
  <div class="summary">
    <table>
      <tr><th>File</th><th>Changes</th></tr>
      <tr><td>src/middleware/auth.ts</td><td>+42 / −18</td></tr>
      <tr><td>src/services/token.ts</td><td>+85 / −3</td></tr>
      <tr><td>src/services/refresh.ts</td><td>+67 / −0 (new)</td></tr>
    </table>
    <p><strong>Primary entry points touched:</strong> <code>authMiddleware()</code>, <code>validateToken()</code></p>
    <p><strong>External touchpoints:</strong> <code>session_store</code> table, Auth Provider HTTP API</p>
  </div>

  <h2>Changed Flow</h2>
  <div class="mermaid">
flowchart LR
  subgraph "Context (unchanged)"
    A([Request]) --> B[Router]
  end

  subgraph "Changed (this branch)"
    B ==> C[Auth Middleware (modified)]
    C ==> D{Token Valid? (modified)}
    D -->|Yes| E[Extract Claims]
    D -->|No| F[Refresh Flow (new)]
  end

  F -.->|HTTP refresh| G{{Auth Provider}}
  F --> H[(session_store)]
  E --> I[Route Handler]
  </div>

  <div class="notes">
    <h3>Notes</h3>
    <ul>
      <li>Refresh flow (<code>src/services/refresh.ts</code>) is entirely new in this branch</li>
      <li>Token validation logic moved from middleware to <code>token.ts</code> service</li>
      <li>Verify: refresh token rotation is handled (check <code>refresh.ts:45</code>)</li>
    </ul>
  </div>

  <script>mermaid.initialize({ startOnLoad: true });</script>
</body>
</html>
```

### Markdown (with `--md` flag, `--branch` mode)

`context/diagrams/branch-oauth-refresh-dataflow.md`:

```markdown
# Data Flow: OAuth Refresh (Branch Changes)

## Scope

- **Branch**: `feature/oauth-refresh`
- **Base ref**: `a1b2c3d` (merge-base of `origin/main`)
- **Included**: Auth middleware → token validation → refresh flow
- **Excluded**: Route handlers, unrelated middleware, session cleanup cron

## Change Summary

| File | Changes |
|------|---------|
| `src/middleware/auth.ts` | +42 / −18 |
| `src/services/token.ts` | +85 / −3 |
| `src/services/refresh.ts` | +67 / −0 (new) |

**Primary entry points touched**: `authMiddleware()`, `validateToken()`
**External touchpoints**: `session_store` table, Auth Provider HTTP API

## Changed Flow

```mermaid
flowchart LR
  subgraph "Context (unchanged)"
    A([Request]) --> B[Router]
  end

  subgraph "Changed (this branch)"
    B ==> C[Auth Middleware (modified)]
    C ==> D{Token Valid? (modified)}
    D -->|Yes| E[Extract Claims]
    D -->|No| F[Refresh Flow (new)]
  end

  F -.->|HTTP refresh| G{{Auth Provider}}
  F --> H[(session_store)]
  E --> I[Route Handler]
```

## Notes

- Refresh flow (`src/services/refresh.ts`) is entirely new in this branch
- Token validation logic moved from middleware to `token.ts` service
- Verify: refresh token rotation is handled (check `refresh.ts:45`)
```

### HTML (default, `--system` mode — unchanged from v1)

`context/diagrams/system-auth-flow.html`:

```html
<!DOCTYPE html>
<html>
<head>
  <title>System: Auth Flow</title>
  <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 1200px; margin: 0 auto; padding: 2rem; }
    h1 { border-bottom: 1px solid #ccc; padding-bottom: 0.5rem; }
    .mermaid { margin: 2rem 0; }
    .notes { background: #f5f5f5; padding: 1rem; border-radius: 4px; margin-top: 2rem; }
  </style>
</head>
<body>
  <h1>System: Auth Flow</h1>
  <p><strong>Scope:</strong> Full authentication flow from request to route handler.</p>

  <div class="mermaid">
flowchart LR
    subgraph Client
        A([Request]) --> B[Auth Middleware]
    end
    subgraph "Auth Service"
        B --> C{Token Valid?}
        C -->|Yes| D[Extract Claims]
        C -->|No| E{Refresh Token?}
        E -->|Yes| F[Refresh Flow]
        E -->|No| G([401 Unauthorized])
        F --> D
    end
    D --> H[Route Handler]
  </div>

  <div class="notes">
    <h3>Notes</h3>
    <ul>
      <li>Simplified: error handling and logging omitted for clarity</li>
    </ul>
  </div>

  <script>mermaid.initialize({ startOnLoad: true });</script>
</body>
</html>
```

## Final Checklist

Before outputting:
- [ ] Diagram is readable without zooming
- [ ] All nodes are reachable (no orphans)
- [ ] Primary path is visually distinct
- [ ] File/function references are accurate and verifiable
- [ ] Complexity matches the scope (don't over-simplify branch changes, don't over-detail high-level system views)
- [ ] **(`--branch` only)** Every changed node has a diff-aware label: `(new)`, `(modified)`, `(moved)`, or `(deleted)`
- [ ] **(`--branch` only)** Changed and context nodes are in separate subgraphs
- [ ] **(`--branch` only)** `==>` edges trace the changed path; `-->` for context; `-.->` for async/optional
- [ ] **(`--branch` only)** Edge labels exist at system boundaries (API, DB, external services)
- [ ] **(`--branch` only)** Change summary section is populated with file stats, entry points, and external touchpoints
- [ ] **(`--branch` only)** Scope statement lists what's included and excluded

Now execute based on the user request above.
