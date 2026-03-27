# Mermaid HTML Templates

## System/Freeform HTML Template

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

## Branch Mode HTML Template

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
      <!-- FILE_ROWS: <tr><td>path/to/file</td><td>+NN / -NN</td></tr> -->
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

## Example: Branch Mode HTML

`_scratch/_diagrams/branch-oauth-refresh-dataflow.html`:

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
    <strong>Included:</strong> Auth middleware -> token validation -> refresh flow<br>
    <strong>Excluded:</strong> Route handlers, unrelated middleware, session cleanup cron
  </div>

  <h2>Change Summary</h2>
  <div class="summary">
    <table>
      <tr><th>File</th><th>Changes</th></tr>
      <tr><td>src/middleware/auth.ts</td><td>+42 / -18</td></tr>
      <tr><td>src/services/token.ts</td><td>+85 / -3</td></tr>
      <tr><td>src/services/refresh.ts</td><td>+67 / -0 (new)</td></tr>
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

## Example: Branch Mode Markdown

`_scratch/_diagrams/branch-oauth-refresh-dataflow.md`:

```markdown
# Data Flow: OAuth Refresh (Branch Changes)

## Scope

- **Branch**: `feature/oauth-refresh`
- **Base ref**: `a1b2c3d` (merge-base of `origin/main`)
- **Included**: Auth middleware -> token validation -> refresh flow
- **Excluded**: Route handlers, unrelated middleware, session cleanup cron

## Change Summary

| File | Changes |
|------|---------|
| `src/middleware/auth.ts` | +42 / -18 |
| `src/services/token.ts` | +85 / -3 |
| `src/services/refresh.ts` | +67 / -0 (new) |

**Primary entry points touched**: `authMiddleware()`, `validateToken()`
**External touchpoints**: `session_store` table, Auth Provider HTTP API

## Changed Flow

` ` `mermaid
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
` ` `

## Notes

- Refresh flow (`src/services/refresh.ts`) is entirely new in this branch
- Token validation logic moved from middleware to `token.ts` service
- Verify: refresh token rotation is handled (check `refresh.ts:45`)
```

## Example: System Mode HTML

`_scratch/_diagrams/system-auth-flow.html`:

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
