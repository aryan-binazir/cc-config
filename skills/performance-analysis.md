---
name: performance-analysis
description: Profiling workflows, benchmark interpretation, and optimization patterns. Identify bottlenecks and apply systematic improvements.
author: Performance Engineering Specialist
version: "1.0"
category: performance
---

# Performance Analysis Skill

Identify, measure, and resolve performance issues systematically. Profile first, optimize second.

## Core Principles

1. **Measure First**: Never optimize without profiling data
2. **Systematic**: Follow methodology, don't randomly tweak
3. **Cost-Benefit**: Engineering time vs performance gains
4. **Real-World**: Optimize for actual usage, not microbenchmarks

## Workflow

### 1. Define Goals

**Requirements**:
- Latency targets (p50, p95, p99)
- Throughput (requests/sec)
- Resource constraints (CPU, memory, disk)
- Scale targets (users, data volume)

**Baseline**: Current performance, target performance, how to measure in production

### 2. Profile

**CPU Profiling Tools**:
| Language | Tools |
|----------|-------|
| Python | cProfile, py-spy, Austin |
| JS/Node | Chrome DevTools, clinic.js |
| Java | JProfiler, YourKit, async-profiler |
| Go | pprof, trace |
| Rust | perf, cargo-flamegraph |

**Look for**: Hot paths, unexpected calls, tight loops, inefficient algorithms (O(n^2))

**Memory Profiling Tools**:
| Language | Tools |
|----------|-------|
| Python | memory_profiler, tracemalloc |
| JS | Chrome DevTools heap profiler |
| Java | JVisualVM, Eclipse MAT |
| Go | pprof (heap) |

**Look for**: Leaks, large allocations, excessive object creation, unbounded caches

**I/O Profiling**: iostat, iotop (Linux), fs_usage (macOS)
**Look for**: Blocking I/O, N+1 patterns, unnecessary operations, missing parallelization

**Database**: EXPLAIN ANALYZE (Postgres), EXPLAIN (MySQL), profiler (MongoDB)
**Look for**: Missing indexes, full table scans, inefficient joins, lock contention

### 3. Interpret Results

**Flame Graphs**:
- Width = time proportion
- Height = call depth
- Wide plateaus = CPU-intensive (investigate)
- Tall towers = deep stacks (possible recursion issue)

**Benchmark Stats**:
- p50: Median (typical experience)
- p95/p99: Tail latency (worst common case)
- Watch for: JIT warmup, cache effects, GC pauses, coordinated omission

### 4. Common Anti-Patterns

```python
# Accidental O(n^2)
for item in items:
    if item in other_list:  # BAD: linear search
        process(item)
# FIX: other_set = set(other_list)

# N+1 queries
for user in users:
    orders = db.query(f"SELECT * FROM orders WHERE user_id = {user.id}")  # BAD
# FIX: Use JOIN or batch query

# Premature materialization
all_data = [process(item) for item in huge_dataset]  # BAD: loads all into memory
# FIX: Use generator: sum(process(item) for item in huge_dataset)

# Lock contention
with global_lock:
    process_user_data(user)  # BAD: coarse locking
# FIX: Fine-grained locks: with user_locks[user.id]:

# Sequential I/O in hot path
const user = await fetchUser(id);
const orders = await fetchOrders(user.id);  // BAD: sequential
# FIX: Promise.all([fetchUser(id), fetchOrders(id), ...])
```

### 5. Optimization Patterns

**Quick Wins**:
1. Add missing indexes (profile queries first)
2. Cache expensive operations (with invalidation strategy)
3. Batch operations (DB inserts, API calls)
4. Use appropriate data structures (sets for membership, heaps for priority)
5. Lazy loading / streaming for large data

**Language-Specific**:
- **Python**: List comprehensions, set/dict lookups, NumPy for numerics, `__slots__`
- **JS/Node**: Don't block event loop, use streaming, worker threads for CPU work
- **Java**: StringBuilder, primitives over wrappers, tune GC, connection pooling
- **Go**: Goroutines (don't overdo), sync.Pool, avoid allocations, buffer channels
- **Rust**: References over copies, reserve Vec capacity, use iterators

### 6. Validate

**Before/After**:
```
Metric              Before    After    Change
-------------------------------------------------
p50 latency         250ms     45ms     -82%
p99 latency         890ms     180ms    -80%
Throughput          100/s     450/s    +350%
```

**Verify**: Correctness preserved, edge cases work, production-like load tested

## Report Template

```markdown
# Performance Analysis: {System}

## Summary
- Current: {metrics}
- Target: {goals}
- Top bottlenecks: {1, 2, 3}
- Recommendations: {prioritized}

## Methodology
Tools: {list}, Environment: {prod-like?}, Load: {rate, concurrency}, Duration: {time}

## Findings

### 1. {Bottleneck}
**Impact**: {% of time/resources}
**Location**: {file:line}
**Root Cause**: {why slow}
**Evidence**: {profile output or metrics}

## Recommendations

### High Priority
1. **{Optimization}**
   - Impact: {-50% latency}
   - Effort: {days}
   - Risk: {low/med/high}
```

## Quick Commands

```bash
# Python
python -m cProfile -o profile.prof script.py

# Node
node --prof app.js && node --prof-process isolate-*.log

# Go
# import _ "net/http/pprof"
go tool pprof http://localhost:6060/debug/pprof/profile

# PostgreSQL
EXPLAIN (ANALYZE, BUFFERS) SELECT ...;
```

## Red Flags

- Single run instead of statistical average
- Microbenchmarks not reflecting real usage
- Testing on dev machine vs prod-like env
- Warm cache assumed when prod has cold cache
- Ignoring tail latencies (p99, p999)

## Stop When

- Goals achieved
- Cost of optimization > benefit
- At hardware limits
- Diminishing returns (80/20 rule hit)
