---
name: mermaid
description: Generate Mermaid diagrams for system understanding
version: "1.0"
argument-hint: [--branch | --system <target> | <description>]
---

# Mermaid Diagram Generator

Generate clear, verifiable Mermaid diagrams that balance high-level understanding with enough detail to confirm correctness.

## User Request

$ARGUMENTS

If `$ARGUMENTS` is empty, ask: "What should I diagram? Options: `--branch` (changes on this branch), `--system <target>` (specific system/flow), or describe what you want to visualize."

## Modes

- `--branch`: Analyze changes on current branch vs main. Produce data flow diagram showing how data moves through the changed code.
- `--system <target>`: Analyze a specific system, module, or flow. `<target>` can be a file path, directory, function name, or concept (e.g., "auth flow", "payment processing").
- **Freeform**: Any other input is treated as a description of what to diagram.

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

**Edge labels:**
- Label edges with data being passed or condition
- Use dotted lines `-.->` for optional/async paths
- Use thick lines `==>` for primary/happy path

### 5. Output

Generate an **HTML file** that renders diagrams directly in the browser.

**Output location**: `context/diagrams/<name>.html`

Where `<name>` is derived from the mode:
- `--branch`: `branch-<branch-name>-dataflow.html`
- `--system`: `system-<target-slug>.html`
- Freeform: `diagram-<slug>.html`

## HTML Template

Use this structure for the output file:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title><!-- DIAGRAM TITLE --></title>
    <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem;
            background: #fafafa;
        }
        h1 { color: #1a1a1a; border-bottom: 2px solid #e0e0e0; padding-bottom: 0.5rem; }
        h2 { color: #333; margin-top: 2rem; }
        .scope { background: #e8f4fd; padding: 1rem; border-radius: 6px; margin: 1rem 0; }
        .diagram-container { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); margin: 1.5rem 0; }
        .notes { background: #fff8e6; padding: 1rem; border-radius: 6px; margin-top: 2rem; }
        .notes ul { margin: 0.5rem 0; padding-left: 1.5rem; }
        .legend { font-size: 0.9rem; color: #666; margin-top: 1rem; }
    </style>
</head>
<body>
    <h1><!-- TITLE --></h1>

    <div class="scope">
        <strong>Scope:</strong> <!-- SCOPE DESCRIPTION -->
    </div>

    <h2><!-- SECTION NAME --></h2>
    <div class="diagram-container">
        <pre class="mermaid">
<!-- MERMAID DIAGRAM CODE -->
        </pre>
    </div>

    <!-- Repeat diagram sections as needed -->

    <div class="notes">
        <strong>Notes:</strong>
        <ul>
            <!-- <li>Note 1</li> -->
        </ul>
    </div>

    <script>mermaid.initialize({ startOnLoad: true, theme: 'default' });</script>
</body>
</html>
```

## Example Output

For a branch analysis, the file `context/diagrams/branch-feature-oauth-refresh-dataflow.html` would contain:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Data Flow: Authentication Changes</title>
    <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem;
            background: #fafafa;
        }
        h1 { color: #1a1a1a; border-bottom: 2px solid #e0e0e0; padding-bottom: 0.5rem; }
        h2 { color: #333; margin-top: 2rem; }
        .scope { background: #e8f4fd; padding: 1rem; border-radius: 6px; margin: 1rem 0; }
        .diagram-container { background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); margin: 1.5rem 0; }
        .notes { background: #fff8e6; padding: 1rem; border-radius: 6px; margin-top: 2rem; }
        .notes ul { margin: 0.5rem 0; padding-left: 1.5rem; }
    </style>
</head>
<body>
    <h1>Data Flow: Authentication Changes</h1>

    <div class="scope">
        <strong>Scope:</strong> Changes introduced in <code>feature/oauth-refresh</code> branch affecting auth flow.
    </div>

    <h2>Request Flow</h2>
    <div class="diagram-container">
        <pre class="mermaid">
flowchart LR
    subgraph Client
        A([Request]) --> B[Auth Middleware]
    end

    subgraph "Auth Service (modified)"
        B --> C{Token Valid?}
        C -->|Yes| D[Extract Claims]
        C -->|No| E{Refresh Token?}
        E -->|Yes| F[Refresh Flow]
        E -->|No| G([401 Unauthorized])
        F --> D
    end

    D --> H[Route Handler]
        </pre>
    </div>

    <div class="notes">
        <strong>Notes:</strong>
        <ul>
            <li>Refresh flow is new in this branch</li>
            <li>Token validation logic moved from middleware to service</li>
        </ul>
    </div>

    <script>mermaid.initialize({ startOnLoad: true, theme: 'default' });</script>
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

Now execute based on the user request above.
