---
name: find-go-mistakes
description: Find common Go mistakes — correctness, concurrency, resource, and performance traps. Use when reviewing, auditing, or fixing Go code.
---

# Find Go Mistakes

Hunt concrete, reachable mistakes in the spirit of *100 Go Mistakes*. The checklist is coverage, not a script.

## Workflow

1. **Map.** Modules and workspaces (`go env GOWORK`, `go list -m`, `go list ./...`), package boundaries, generated code, tests, and the project's rules and validation commands (README, Makefile, CI, `AGENTS.md`/`CLAUDE.md`).
2. **Discover.** Dispatch one read-only sub-agent per Checklist section, in parallel, each with the same scope, map, and rules, using the prompt below. Without sub-agents, run the same section-by-section passes yourself and say the review was single-agent.
3. **Verify.** Merge and dedupe candidates. Read call sites before filing: confirm the path is reachable, callers don't already enforce the contract, and tests don't already cover it.
4. **Report** in the Finding Format. Review is report-only.
5. **Patch** only what the user picks — a finding, a file, or a scoped class of issues — narrowly, preserving public behavior unless they approve a change. Then run Verification.

## Discovery Sub-Agent Prompt

```text
Review this Go repository for common Go mistakes in the assigned section only. Report; do not patch.

Repo scope: <packages/files/modules under review>
Project rules: <relevant AGENTS/CLAUDE/README/CI constraints>
Assigned section: <checklist section name and its bullets>

For each candidate finding:
- Severity: Critical | High | Medium | Low
- Confidence: High | Medium | Low
- Location: path/to/file.go:line
- Evidence: concrete code path or behavior
- Suggested fix: smallest practical fix

No concrete candidates: say "No findings."
```

## Finding Format

Ordered by severity:

```md
## Findings
- Severity: Critical | High | Medium | Low
  Confidence: High | Medium | Low
  Location: path/to/file.go:123
  Issue: the Go mistake or risk.
  Why it matters: the failing scenario or operational cost.
  Suggested fix: smallest practical fix, with test coverage when useful.

## Uncertain
Same shape, for items with incomplete evidence.

## No Findings
Say so plainly; name important test gaps or unexercised areas.

## Patch Prompt
Ask: "Which finding do you want me to patch?"
```

## Checklist

### Organization and API Shape
- Shadowing, deep nesting, `init` overuse, package-level side effects, hidden startup-order coupling.
- Interface pollution: premature or producer-side interfaces, returning interfaces, vague `any`; getter/setter boilerplate.
- Generics, embedding, reflection, `unsafe`, or cgo where concrete code is clearer or safer.
- Config APIs that can't grow without breaking callers (functional options only where they relieve real pressure).
- Grab-bag utility packages, package-name collisions, undocumented exports, ignored lint/staticcheck signals.

### Data Types and Values
- Leading-zero (octal) literals, integer overflow, float equality/rounding, unsafe numeric conversions.
- Slices: len/cap confusion, missed preallocation, nil-vs-empty contract drift, wrong empty checks, bad copies, `append` aliasing, sub-slices pinning large backing arrays.
- Maps: missed preallocation, memory retained after deletes, order-dependent iteration, mutation during iteration, nil-map writes.
- Invalid comparisons, typed nil inside interfaces, nil receivers, ambiguous zero values, internal slices/maps returned uncopied.

### Control Flow, Strings, and Functions
- `range` value copies, pointers to range values, range expressions evaluated once, `break` hitting `switch`/`select` instead of the loop.
- `defer` in unbounded loops, defer arguments/receivers evaluated at defer time, cleanup errors masking the primary error.
- Rune vs byte, Unicode iteration, `Trim` cutset vs `TrimSuffix`, substrings pinning large strings, wasteful conversions, slow string assembly.
- Receivers copying locks or large mutable state; named results that hurt clarity or cause side effects; filename parameters where `io.Reader`/`io.Writer`/`fs.FS` would decouple.

### Errors and Control Flow
- `panic` for expected errors, `recover` hiding failure, `log.Fatal`/`os.Exit` in reusable code.
- Wrapping that breaks `errors.Is`/`errors.As`; `==` or type checks that fail once wrapped; inaccurate sentinel comparisons.
- Errors handled twice; dropped errors from writes, scanners, encoders, `sql.Rows`, and cleanup/`Close` where correctness depends on them.
- Partial state mutated before an error return; shadowed variables returning stale values.

### Concurrency and Context
- Concurrency that can't raise throughput, or where the workload type lets scheduling overhead dominate.
- Channel vs mutex mismatched to ownership; channel buffer sizes with no blocking/backpressure reason.
- Goroutines lacking a stop path, cancellation, error propagation, panic containment, or bounded lifetime.
- Loop variables captured by goroutines or deferred closures; `select` assumed deterministic.
- Notification and nil channels, `sync.Cond`, `errgroup`, `WaitGroup`: misused, or missed where they fit.
- Races on shared state, `append`, slices/maps, logging/formatting side effects, copied `sync` values, test helpers.
- Context misuse: propagated into background work that should detach, missing from I/O that needs it, used as a parameter bag.

### Standard Library and Resources
- Wrong time units, timer/ticker leaks, `time.After` in loops, missing `Stop`, stale timer values, brittle time comparisons.
- JSON surprises: embedding, monotonic clock fields, `map[string]any` numeric precision, unknown fields.
- SQL: `sql.Open` doesn't connect, ignored pool behavior, missing prepared statements, mishandled NULLs, unchecked row-iteration errors.
- HTTP: handlers that write an error and keep executing, default clients/servers without timeouts, unsafe transport defaults.
- Unclosed transient resources: HTTP bodies, `sql.Rows`, files, sockets, locks, transactions, statements.

### Tests and Benchmarks
- Tests not split by unit/integration/slow/external dependency via build tags, env gates, or `testing.Short`.
- Race-prone code without `-race` coverage, order-dependent tests, `-shuffle`/parallel coverage missing where useful, sleeps instead of synchronization.
- Missing table-driven tests where cases matter, no external-package tests for public APIs, weak setup/teardown, unused `httptest`, `iotest`, `testing/fstest`, `t.TempDir`.
- Benchmarks timing setup, missing timer reset/pause, no allocation reporting, results optimized away, overfit microbenchmarks, observer effects.

### Performance and Runtime
- Hot-path waste: allocations, conversions, formatting, reflection, per-call regex/parser setup, concatenation, buffering, copying.
- N+1 database/API calls, unbounded concurrency, missing backpressure, unbounded caches.
- Mechanical sympathy in performance-sensitive code: cache-unfriendly layout, false sharing, alignment waste, branch unpredictability, missed instruction-level parallelism.
- Stack/heap misreads, avoidable escapes, missed inlining, `sync.Pool` misuse, optimization claims without profiler/trace evidence.
- Runtime vs deployment: GC pressure, `GOMAXPROCS`, CPU/memory limits, container/Kubernetes resource assumptions.

## Verification

After patching, run the narrowest useful checks first, preferring stricter project commands:

```bash
go test ./path/to/package
go test ./...
```

Add `-race` when the change touches goroutines, shared memory, channels, timers, caches, handlers, background workers, or parallel tests:

```bash
go test -race ./path/to/package
```

Benchmark only performance findings or patches that could materially move a hot path:

```bash
go test -bench 'BenchmarkName' -benchmem ./path/to/package
```

When validation is expensive or needs services, run the relevant subset and state exactly what was and wasn't exercised.
