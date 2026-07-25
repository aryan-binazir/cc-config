---
name: find-go-mistakes
description: Scan Go repositories for common Go mistakes and Go correctness/performance/concurrency risks before making changes. Use whenever the user asks to review, scan, audit, or fix Go code for common mistakes, bugs, concurrency issues, API misuse, testing gaps, performance traps, maintainability risks, or suspicious Go behavior, especially around goroutines, channels, contexts, error handling, nil behavior, slices/maps, interfaces, defer/cleanup, timeouts, HTTP/database/resource handling, or performance-sensitive paths.
---

# Find Go Mistakes

Review Go codebases for common Go mistakes before changing code. Favor concrete, actionable findings over broad advice. Treat the checklist as coverage guidance, not as text to recite.

## Workflow

1. Inspect the repo before judging:
   - Identify modules and workspaces with `go env GOWORK`, `go list -m`, `go list ./...`, and `find . -name go.mod -o -name go.work` as appropriate.
   - Map package boundaries, command packages, generated files, tests, internal packages, and integration-heavy areas.
   - Check available project rules and normal validation commands from `README`, `Makefile`, CI config, `AGENTS.md`, `CLAUDE.md`, and similar files.
2. Search for risk surfaces:
   - Package layout, API shape, interfaces, exported names, documentation, lint coverage, and confusing utility packages.
   - Numeric, string, slice, map, range-loop, receiver, and named-result behavior that can silently change semantics.
   - Goroutines, channels, timers, mutexes, atomics, `context.Context`, `errgroup`, `sync.Once`, `WaitGroup`, worker pools, and shutdown paths.
   - Error handling, wrapping, sentinel errors, ignored returns, panic/recover, deferred cleanup, and partial writes.
   - HTTP clients/servers, JSON, SQL, time APIs, databases, transactions, rows/results, files, sockets, body closing, timeouts, retries, and resource ownership.
   - Tests, benchmarks, race coverage, time-dependent tests, table-driven tests, and package-level test seams.
   - Hot paths, allocation-heavy code, serialization, caches, SQL loops, runtime diagnostics, deployment limits, and benchmarks.
3. Spawn parallel discovery sub-agents, one per checklist section:
   - `Organization and API Shape`
   - `Data Types and Values`
   - `Control Flow, Strings, and Functions`
   - `Errors and Control Flow`
   - `Concurrency and Context`
   - `Standard Library and Resources`
   - `Tests and Benchmarks`
   - `Performance and Runtime`
4. Give every discovery sub-agent the same repo scope, module/package map, project rules, and validation constraints. Tell each sub-agent to inspect only its assigned section and return candidate findings with `path:line`, severity, confidence, evidence, and suggested fix. Discovery sub-agents must not patch.
5. As the main agent, merge the discovery results, remove duplicates, and verify candidates before reporting. Read call sites before filing a finding. Confirm whether an apparent issue is reachable, whether callers enforce a contract, and whether tests already cover it.
6. If sub-agents are unavailable, run the same section-by-section discovery passes yourself and state that the review was single-agent.
7. Produce findings first. Do not patch during the review pass.
8. Ask the user which findings they want patched. If they choose one, make the smallest focused change and run relevant Go tests.

## Discovery Sub-Agent Prompt

Use this shape for each checklist section:

```text
Review this Go repository for common Go mistakes in the assigned section only.

Repo scope: <packages/files/modules under review>
Project rules: <relevant AGENTS/CLAUDE/README/CI constraints>
Assigned section: <one checklist section name>

Return only candidate findings. Do not patch.
For each candidate include:
- Severity: Critical | High | Medium | Low
- Confidence: High | Medium | Low
- Location: path/to/file.go:line
- Evidence: concrete code path or behavior
- Suggested fix: smallest practical fix

If there are no concrete candidates for this section, say "No findings."
```

## Finding Format

Report findings first, ordered by severity. Use this shape:

```md
## Findings
- Severity: Critical | High | Medium | Low
  Confidence: High | Medium | Low
  Location: path/to/file.go:123
  Issue: concise statement of the common Go mistake or Go correctness/performance/concurrency risk.
  Why it matters: explain the failing scenario or operational cost.
  Suggested fix: smallest practical fix, including test coverage when useful.

## No Findings
Say plainly when no concrete issues were found. Mention important test gaps or areas not exercised.

## Patch Prompt
Ask: "Which finding do you want me to patch?"
```

Use exact file/line references. If evidence is incomplete, put the item under `Uncertain` instead of overstating it.

## Patch Permission Rule

Do not patch automatically after a scan or review. A review request means report findings only.

Patch only when the user asks to fix a specific finding, a specific file, or a clearly scoped class of issues. Keep the patch narrow, avoid drive-by refactors, and preserve public behavior unless the user explicitly approves a behavior change.

## Checklist

### Organization and API Shape
- Shadowed variables, unnecessary nesting, overused `init`, package-level side effects, and hidden startup order dependencies.
- Getter/setter boilerplate, premature interfaces, producer-owned interfaces, returning interfaces instead of concrete values, and vague `any` usage.
- Generics, type embedding, reflection, unsafe, or cgo used where simpler concrete code would be clearer or safer.
- Config APIs that would be hard to extend without compatibility breaks; consider functional options only when it reduces real API pressure.
- Confusing package layout, grab-bag utility packages, package name collisions, missing exported docs, and ignored lint/staticcheck signals.

### Data Types and Values
- Leading-zero numeric literals, integer overflow, float equality/rounding assumptions, and unsafe numeric conversions.
- Slice length/capacity confusion, poor preallocation, nil-vs-empty slice contract drift, incorrect empty checks, bad copies, append aliasing, and retained large backing arrays.
- Map preallocation misses, maps that retain memory after deletes, order-dependent iteration, mutation-during-iteration surprises, and nil map writes.
- Invalid comparisons, typed nils inside interfaces, nil receivers, ambiguous zero values, and mutable slices/maps returned without copying.

### Control Flow, Strings, and Functions
- Range-loop value copies, pointer-to-range-value bugs, loop arguments evaluated at surprising times, and `break` targeting the wrong construct.
- `defer` inside unbounded loops, deferred arguments/receivers evaluated earlier than intended, and cleanup that masks the primary error.
- Rune vs byte confusion, incorrect Unicode iteration, trim/cutset confusion, substring retention of large strings, wasteful conversions, and slow string assembly.
- Receiver choices that copy locks or large mutable state, named returns that help or hurt clarity, named-result side effects, and APIs that take filenames where `io.Reader`, `io.Writer`, or `fs.FS` would decouple behavior.

### Errors and Control Flow
- Panics for expected errors, recover that hides failure, and `log.Fatal` or `os.Exit` in reusable code.
- Error wrapping that breaks `errors.Is`/`errors.As`, direct type/value checks that fail after wrapping, and sentinel errors compared inaccurately.
- Handling the same error twice, dropping errors from writes/scanners/encoders/SQL rows/cleanup, and missing close/defer error handling when it affects correctness.
- Partial state updates before returning an error and shadowed variables that return stale values.

### Concurrency and Context
- Concurrency added where it cannot improve throughput or where the workload type makes scheduling overhead dominate.
- Channels used where a mutex is clearer, mutexes used where ownership transfer would be clearer, and channel buffering chosen without a blocking/backpressure reason.
- Goroutines without a known stop path, cancellation, error propagation, panic containment, or bounded lifetime.
- Loop variables captured by goroutines or deferred closures; `select`/channel behavior assumed deterministic.
- Notification channels, nil channels, `sync.Cond`, `errgroup`, and `WaitGroup` used incorrectly or missed where they fit.
- Data races around shared state, append, slices/maps, logging/string formatting side effects, copied sync values, and test helpers.
- Contexts propagated into background work when they should be detached, not propagated through I/O when they should be, or used as arbitrary parameter bags.

### Standard Library and Resources
- Wrong time units, timer/ticker leaks, `time.After` in loops, missing `Stop`, stale timer values, and brittle time comparisons.
- JSON surprises from embedding, monotonic time fields, maps of `any`, missing unknown-field handling, or numeric precision assumptions.
- SQL mistakes: assuming `sql.Open` connects, ignoring pool behavior, skipping prepared statements where needed, mishandling nulls, and missing row iteration errors.
- HTTP handlers that write an error response and keep executing, default clients/servers with no timeouts, and unsafe transport defaults.
- Unclosed transient resources: HTTP bodies, `sql.Rows`, files, sockets, locks, transactions, and statements.

### Tests and Benchmarks
- Tests not categorized by unit/integration/slow/external dependency; missing build tags, env gates, or `testing.Short` behavior.
- Race-prone code without race-test coverage, tests that depend on order, missing `-shuffle`/parallel coverage where useful, and sleeps instead of synchronization.
- Missing table-driven tests where cases matter, no external-package tests for public APIs, weak setup/teardown, and missed helpers from `httptest`, `iotest`, `testing/fstest`, or `t.TempDir`.
- Benchmarks that include setup, fail to reset/pause timers, omit allocation reporting, get optimized away, overfit microbenchmarks, or ignore observer effects.

### Performance and Runtime
- Hot paths with unnecessary allocations, conversions, formatting, reflection, regex/parser setup, string concatenation, buffering, or copying.
- N+1 database/API calls, unbounded concurrency, missing backpressure, and cache growth without limits.
- Cache-unfriendly data layout, false sharing, alignment waste, branch predictability problems, and missed instruction-level parallelism in performance-sensitive code.
- Misread stack vs heap behavior, avoidable escapes, missed inlining opportunities, `sync.Pool` misuse, and no profiler/trace evidence for optimization claims.
- Runtime/deployment mismatches: GC pressure, `GOMAXPROCS`, CPU/memory limits, and container/Kubernetes resource assumptions.

## Verification

After patching, run the narrowest useful checks first:

```bash
go test ./path/to/package
go test ./...
```

Use race tests when the change touches goroutines, shared memory, channels, timers, caches, handlers, background workers, or tests with parallelism:

```bash
go test -race ./path/to/package
```

Run targeted benchmarks only for performance findings or when a patch could materially affect a hot path:

```bash
go test -bench 'BenchmarkName' -benchmem ./path/to/package
```

Prefer project-provided commands when they are stricter than raw `go test`. If validation is expensive or requires services, run the relevant subset and say exactly what was and was not exercised.
