---
name: mermaid
description: Generate clear Mermaid diagrams for a branch diff, a specific system or flow, or a freeform architecture request, and save the result under `_scratch/_diagrams/`. Use when the user asks for a Mermaid diagram, system visualization, branch data-flow map, or architecture diagram.
---

# Mermaid

Generate verifiable Mermaid diagrams that balance high-level understanding with enough implementation detail to confirm correctness.

## Modes

Infer the mode from the request:
- **branch mode**: the user wants a diagram of current branch changes
- **system mode**: the user names a file, directory, function, module, or flow
- **freeform mode**: the user describes what to visualize more generally

If the request is ambiguous, ask: "What should I diagram? Options: branch changes, a specific system or flow, or a freeform description."

If the user wants markdown output in addition to HTML, produce both.

## Workflow

### 1. Gather Context

**Branch mode:**
```bash
BASE=$(git merge-base origin/main HEAD 2>/dev/null || git merge-base origin/master HEAD)
git diff --stat $BASE..HEAD
git diff $BASE..HEAD
```

**System mode:**
- Locate relevant files using grep/glob
- Read entry points, core logic, and data models
- Trace the flow through the system

**Freeform mode:**
- Parse the description to understand what needs diagramming
- Gather relevant code context

### 2. Analyze & Design

Identify:
- **Entry points**: Where data/control enters
- **Transformations**: What happens to data at each step
- **Decision points**: Branches, conditions, error paths
- **Exit points**: Where data/control leaves (returns, side effects, external calls)
- **Key actors**: Services, modules, functions, external systems

Additional analysis for branch mode:
- **Changed nodes**: Functions, modules, or components added, modified, moved, or deleted
- **Primary entry points touched**: Entry points whose behavior changed in this branch
- **External touchpoints affected**: DB tables, external services, APIs, emitted events impacted by the diff
- **Context nodes**: Minimal set of unchanged nodes required to make the changed flow understandable

### 3. Select Diagram Type

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

### 5. Branch-Specific Diagram Rules (branch mode only)

These rules apply on top of the base style rules.

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

- `subgraph "Changed (this branch)"` -- all nodes that were added, modified, or moved
- `subgraph "Context (unchanged)"` -- minimal surrounding nodes needed to make the flow understandable

Prioritize the changed flow. Include the fewest unchanged nodes necessary for comprehension. The 15-20 node cap still applies; split into multiple diagrams if needed.

#### 5c. Edge Semantics (strict)

| Edge | Meaning |
|------|---------|
| `==>` | Primary changed path -- the flow reviewers should follow |
| `-->` | Synchronous calls providing context (unchanged paths, supporting calls) |
| `-.->` | Async, optional, stream, or event-driven edges |

Add edge labels at system boundaries where reviewers care most: API/input boundary, DB boundary, external API boundary, emitted events.

#### 5d. Optional Impact Map

When the diff spans multiple domains or architectural layers, generate a second diagram (max 10 nodes) showing:

```
Changed files/packages -> Impacted modules -> External systems
```

Only include this when it adds clarity beyond the main diagram. Skip it for single-domain changes.

#### 5e. Scope Statement

Explicitly state what the diagram covers:
- **Included**: Flows that traverse changed code, plus minimal context
- **Excluded**: Untouched subsystems, unrelated files in the diff (if any), internal implementation detail not required for verification

## Output

**Output location**: `_scratch/_diagrams/<name>.<ext>`

Where `<name>` is derived from the mode:
- branch mode: `branch-<branch-name>-dataflow`
- system mode: `system-<target-slug>`
- freeform: `diagram-<slug>`

### Default: HTML Output (.html)

Generate a self-contained HTML file that renders the Mermaid diagram in browser. Follow the HTML templates in `references/templates.md`. Use the branch-mode template for branch diagrams and the system/freeform template for everything else.

### With Markdown: Additional Markdown Output (.md)

When the user requests markdown too, also generate a markdown file.

**For system and freeform modes:**

1. **Title**: What this diagram shows
2. **Scope**: What's included/excluded
3. **Diagram(s)**: The Mermaid code block(s)
4. **Legend** (if needed): Explain non-obvious symbols
5. **Notes**: Key assumptions, simplifications, or areas needing attention

**For branch mode:**

1. **Title**: `Data Flow: <feature> (Branch Changes)`
2. **Scope**: Base ref, branch, what's included/excluded
3. **Change Summary**: Files changed (+/-), primary entry points touched, DB tables / external systems touched
4. **Diagram 1**: "Changed Flow (with minimal context)"
5. **Diagram 2** (optional): "Impact Map" -- only when diff spans multiple domains
6. **Notes**: Assumptions + verification hints (what to check in code)

## Final Checklist

Before outputting:
- [ ] Diagram is readable without zooming
- [ ] All nodes are reachable (no orphans)
- [ ] Primary path is visually distinct
- [ ] File/function references are accurate and verifiable
- [ ] Complexity matches the scope (don't over-simplify branch changes, don't over-detail high-level system views)
- [ ] **(branch only)** Every changed node has a diff-aware label: `(new)`, `(modified)`, `(moved)`, or `(deleted)`
- [ ] **(branch only)** Changed and context nodes are in separate subgraphs
- [ ] **(branch only)** `==>` edges trace the changed path; `-->` for context; `-.->` for async/optional
- [ ] **(branch only)** Edge labels exist at system boundaries (API, DB, external services)
- [ ] **(branch only)** Change summary section is populated with file stats, entry points, and external touchpoints
- [ ] **(branch only)** Scope statement lists what's included and excluded
