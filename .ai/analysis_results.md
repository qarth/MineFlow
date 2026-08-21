# MineFlow Go Code — Analysis & Improvement Recommendations

Analysis of [mineflow.go](file:///c:/Users/rob/code/MineFlow/mineflow.go) and [mineflow_test.go](file:///c:/Users/rob/code/MineFlow/mineflow_test.go).

> [!NOTE]
> The C++ files (`mineflow.h`, `mineflow.cpp`) are the upstream reference implementation by Matthew Deutsch and are **not** covered in this review. The analysis focuses exclusively on the Go port.

---

## 1. Use `min` Built-in Instead of Custom `min64`

**File:** [mineflow.go#L336-L341](file:///c:/Users/rob/code/MineFlow/mineflow.go#L336-L341)

Go 1.21+ added a built-in generic `min` / `max`. Since `go.mod` specifies `go 1.22`, the custom `min64` helper is unnecessary.

```diff
-                res := dfs(e.to, min64(pushed, e.cap))
+                res := dfs(e.to, min(pushed, e.cap))
```

Delete the entire `min64` function (lines 336–341).

---

## 2. `OneFive` / `OneNine` Should Be Free Functions or Constructors, Not Methods

**File:** [mineflow.go#L104-L116](file:///c:/Users/rob/code/MineFlow/mineflow.go#L104-L116)

`OneFive()` and `OneNine()` are methods on `PrecedencePattern` but ignore the receiver entirely — they construct a brand-new value. The call site in the test even has to create a throwaway `NewPrecedencePattern(nil)` just to call them:

```go
// current — awkward
pattern := NewPrecedencePattern(nil).OneFive()
```

Idiomatic Go would make these package-level constructor functions:

```go
func OneFivePrecedencePattern() PrecedencePattern { ... }
func OneNinePrecedencePattern() PrecedencePattern { ... }
```

---

## 3. `Antecedents` Returns a Defensive Copy — Consider Documenting or Removing the Copy

**File:** [mineflow.go#L49-L57](file:///c:/Users/rob/code/MineFlow/mineflow.go#L49-L57)

`ExplicitPrecedence.Antecedents` allocates a new slice and copies every call. This is safe but expensive in the hot solver loop. Two options:

| Option | Trade-off |
|--------|-----------|
| **Document** that callers must not mutate the returned slice and return the backing slice directly. | Faster, but callers must respect the contract. |
| **Keep the copy** but pre-sort once in `AddConstraint` instead of sorting per-call elsewhere. | Keeps safety, removes redundant work. |

For the `Regular3DBlockModelPatternPrecedence`, `Antecedents` also sorts every call (line 153). Since the pattern offsets are fixed, pre-sorting offsets once at construction time would eliminate repeated work.

---

## 4. `SolveUltimatePit` Silently Swallows Errors

**File:** [mineflow.go#L224-L241](file:///c:/Users/rob/code/MineFlow/mineflow.go#L224-L241)

Both `NewPseudoSolver` and `Solve` can fail, but `SolveUltimatePit` silently returns an all-false slice on error. This makes debugging impossible. Consider:

```go
func SolveUltimatePit(values []int64, precedence [][]int64) ([]bool, error) {
```

If the function must remain error-free for API compatibility, at minimum log a warning.

Also: silently skipping `len(pair) != 2` constraints (line 228) hides malformed input.

---

## 5. `dinic` — Struct-of-Pointers Causes Excessive GC Pressure

**File:** [mineflow.go#L243-L263](file:///c:/Users/rob/code/MineFlow/mineflow.go#L243-L263)

```go
type dinic struct {
    g [][]*edge  // each edge is a separate heap allocation
}
```

Every `addEdge` call creates **two** `*edge` allocations. For large pit problems (hundreds of thousands of blocks) this creates millions of tiny objects for the GC to scan.

**Recommended refactor:** Use a flat edge pool with index-based references:

```go
type edge struct {
    to  int
    cap int64
    rev int   // index into edges slice
}

type dinic struct {
    head  []int   // head[node] = index of first edge, -1 if none
    edges []edge  // flat contiguous pool
}
```

This is the standard competitive-programming pattern and eliminates all pointer chasing.

---

## 6. BFS Queue Uses Slice-Shift Anti-Pattern

**File:** [mineflow.go#L274-L276](file:///c:/Users/rob/code/MineFlow/mineflow.go#L274-L276) and [L323-L325](file:///c:/Users/rob/code/MineFlow/mineflow.go#L323-L325)

```go
queue := []int{source}
for len(queue) > 0 {
    cur := queue[0]
    queue = queue[1:]  // doesn't release memory, grows unbounded
```

`queue[1:]` re-slices without ever shrinking. For large graphs this leaks memory for the lifetime of the BFS. Use a ring buffer or a `container/list`, or at minimum a two-index approach:

```go
queue := []int{source}
for head := 0; head < len(queue); head++ {
    cur := queue[head]
    // ...
}
```

---

## 7. `Solve` Copies `seen` to `inCut` Unnecessarily

**File:** [mineflow.go#L216-L220](file:///c:/Users/rob/code/MineFlow/mineflow.go#L216-L220)

```go
seen := dinic.reachableFrom(source)
inCut := make([]bool, n)
for i := 0; i < n; i++ {
    inCut[i] = seen[i]
}
```

`seen` has length `n+2` (includes source/sink sentinel nodes). You can simply truncate:

```go
return dinic.reachableFrom(source)[:n], nil
```

---

## 8. Magic Number `1<<60` Should Be a Named Constant

**File:** [mineflow.go#L209](file:///c:/Users/rob/code/MineFlow/mineflow.go#L209) and [L309](file:///c:/Users/rob/code/MineFlow/mineflow.go#L309)

```go
dinic.addEdge(from, to, int64(1<<60))  // infinity-capacity edge
```

Use a named constant for clarity and to avoid mismatched values:

```go
const infCapacity int64 = 1 << 60
```

---

## 9. `BlockValues` Interface Is Under-Utilized

**File:** [mineflow.go#L18-L21](file:///c:/Users/rob/code/MineFlow/mineflow.go#L18-L21)

`BlockValues` is defined but `PseudoSolver` stores a `[]int64` internally anyway. The only path from `BlockValues` is [NewPseudoSolverFromValues](file:///c:/Users/rob/code/MineFlow/mineflow.go#L174-L183) which immediately materialises the whole slice. Consider:

- Dropping the interface and accepting `[]int64` everywhere (simplicity), **or**
- Having the solver use the interface lazily (useful if values are computed on-the-fly from a block model).

---

## 10. `ExplicitPrecedence` Uses `map[int][]int` — Consider a Flat `[][]int`

**File:** [mineflow.go#L35-L38](file:///c:/Users/rob/code/MineFlow/mineflow.go#L35-L38)

Since block indices are dense `[0, numBlocks)`, a map adds unnecessary overhead. A flat slice is more idiomatic and faster:

```go
type ExplicitPrecedence struct {
    antecedents [][]int   // indexed by block
}
```

---

## 11. Tests — Use `slices.Equal` and Table-Driven Style

**File:** [mineflow_test.go](file:///c:/Users/rob/code/MineFlow/mineflow_test.go)

The manual element-by-element comparison (lines 11–18) is verbose. Go 1.21+ provides `slices.Equal`:

```go
if !slices.Equal(got, want) {
    t.Fatalf("got %v, want %v", got, want)
}
```

The tests would also benefit from table-driven style to cover edge cases (empty graph, single block, all-negative values, etc.).

---

## 12. `go.mod` Version and Module Path

**File:** [go.mod](file:///c:/Users/rob/code/MineFlow/go.mod)

| Issue | Recommendation |
|-------|----------------|
| Module path `mineflow` is unversioned and non-importable | Use `github.com/MineFlowCSM/MineFlow` (or wherever it's hosted) |
| `go 1.22` without a patch version | Consider specifying the full toolchain e.g. `go 1.22.0` |

---

## 13. Bounds Checking in `Regular3DBlockModelPatternPrecedence.Antecedents`

**File:** [mineflow.go#L147](file:///c:/Users/rob/code/MineFlow/mineflow.go#L147)

The long condition on a single line hurts readability:

```go
if candidateX < 0 || candidateX >= p.blockDef.NumX || candidateY < 0 || candidateY >= p.blockDef.NumY || candidateZ < 0 || candidateZ >= p.blockDef.NumZ {
```

Extract a helper method on `BlockDefinition`:

```go
func (b BlockDefinition) InBounds(x, y, z int) bool {
    return x >= 0 && x < b.NumX && y >= 0 && y < b.NumY && z >= 0 && z < b.NumZ
}
```

This mirrors the C++ `InDef` method and makes every call site cleaner.

---

## Summary — Priority Matrix

| # | Issue | Impact | Effort |
|---|-------|--------|--------|
| 5 | Flat edge pool for Dinic | 🔴 Performance | Medium |
| 6 | BFS queue memory leak | 🔴 Performance | Low |
| 1 | Replace `min64` with built-in | 🟢 Cleanup | Trivial |
| 7 | Eliminate redundant copy | 🟢 Cleanup | Trivial |
| 8 | Named constant for infinity | 🟢 Clarity | Trivial |
| 2 | Pattern constructors → free functions | 🟡 API design | Low |
| 4 | Error handling in `SolveUltimatePit` | 🟡 Correctness | Low |
| 10 | Dense slice vs map | 🟡 Performance | Low |
| 3 | Pre-sort antecedents once | 🟡 Performance | Low |
| 13 | `InBounds` helper | 🟢 Readability | Trivial |
| 11 | `slices.Equal` + table tests | 🟢 Test quality | Low |
| 9 | Simplify `BlockValues` interface | 🟡 API design | Medium |
| 12 | `go.mod` module path | 🟡 Importability | Low |

> [!TIP]
> Items 1, 7, 8, and 13 are quick wins that can be done in a single pass. Item 5 (flat edge pool) would yield the largest performance improvement for large-scale pit problems.

Let me know which items you'd like me to implement, or if you'd like me to tackle all of them with an implementation plan.
