# engine

The `engine` package re-exports some of Insyra's core data structures and algorithms, providing safe, well-tested primitives for external packages to build upon.

## Table of Contents

- [Overview](#overview)
- [Exports](#exports)
  - [BiIndex](#biindex)
  - [Ring](#ring)
  - [CCL](#ccl)
  - [Sorting & Comparison Utilities](#sorting--comparison-utilities)
- [Examples](#examples)
- [Notes](#notes)
- [Related Links](#related-links)

---

## Overview

`engine` aims to expose a small set of highly useful internals from `internal/core` and `internal/algorithms` with a clean API. Typical use cases include a stable id↔name index (`BiIndex`), mixed-type comparison utilities (`CompareAny` / `GetTypeSortingRank`), and a stable parallel sorter for large slices (`ParallelSortStableFunc`).

## Exports

### BiIndex

`BiIndex` is a bidirectional index (id ↔ name) that guarantees stable ids and supports reusing deleted ids via a free list. It is deliberately implemented as non-concurrent; callers should provide synchronization when used from multiple goroutines.

```go
// type alias
type BiIndex = core.BiIndex

// Constructor
func NewBiIndex(cap int) *BiIndex
```

Key methods (see `internal/core/biindex.go` for full details):

- `Assign(name string) (int, bool)` — assign an id for a name (reuses freed ids), returns (id, true) if newly assigned.
- `Set(id int, name string) (string, bool)` — set a specific id to a name, returns previous name and success.
- `Get(id int) (string, bool)` — get name by id.
- `Index(name string) (int, bool)` — get id by name.
- `DeleteByID(id int) (string, bool)` — delete mapping and push id to free list.
- `DeleteAndShift(id int) (string, map[int]int, bool)` — delete the id and shift larger ids down by 1, returning the old→new id map.
- `DeleteByName(name string) bool`, `Has(name string) bool`, `Len() int`, `IDs() []int`, `Clone() *BiIndex`, `Clear()`

Example:

```go
b := NewBiIndex(16)
idA, _ := b.Assign("Alice")
idB, _ := b.Assign("Bob")
name, ok := b.Get(idA) // "Alice", true
b.DeleteByName("Bob")
// idB will be added to free list and may be reused by subsequent Assign calls
```

#### Performance Characteristics ⚡

`BiIndex` is implemented using two Go maps (`idToString` and `stringToID`) and a free-list (`freed`). Map operations are O(1) on average in Go; the free-list is a slice and supports O(1) push/pop but may need O(n) scanning in some operations. Below are the time and space complexity characteristics of common operations:

- `Assign(name string) (int, bool)` — Average: O(1) (map lookup + free-list pop is O(1)).
- `Set(id int, name string) (string, bool)` — Average: O(1); Worst-case: O(n) (scans `freed` slice to remove an id if present).
- `Get(id int) (string, bool)` — O(1) (map lookup).
- `Index(name string) (int, bool)` — O(1) (map lookup).
- `DeleteByID(id int) (string, bool)` — O(1) (map deletes and append to `freed`).
- `DeleteByName(name string) bool` — O(1) (map lookup + delete).
- `DeleteAndShift(id int) (string, map[int]int, bool)` — O(n) (must reassign many ids and adjust maps; shifts are linear in the number of larger ids).
- `Has(name string) bool` — O(1) (map lookup).
- `Len() int` — O(1) (map length).
- `IDs() []int` — O(n) (iterate map keys and create slice).
- `Clone() *BiIndex` — O(n) (deep copy of maps and free list).
- `Clear()` — O(n) (delete all map entries and reset internal slices).

Space complexity: O(n) additional space for the two maps and the free-list where n is the number of registered names/ids.

**Concurrency note:** `BiIndex` is NOT concurrent-safe; provide external synchronization (e.g., `sync.Mutex`) when using across goroutines.

### Ring

`Ring` is a non-thread-safe circular buffer with dynamic growth. It is suitable for building higher-level queues or error rings.

```go
// type alias
type Ring[T any] = core.Ring[T]

// Constructor
func NewRing[T any](capacity int) *Ring[T]
```

Key methods (see `internal/core/ring.go` for full details):

- `Len() int` — number of elements currently in the ring.
- `Get(i int) (T, bool)` — return element at logical index `i` (0..Len-1).
- `ToSlice() []T` — return a copy of the ring contents in logical order.
- `Clear()` — remove all elements while keeping capacity.
- `Push(v T)` — add an element to the back of the ring.
- `PopFront() (T, bool)` — remove and return the front element.
- `PopBack() (T, bool)` — remove and return the last element.
- `DeleteAt(idx int) (T, bool)` — remove the element at logical index `idx`.

**Concurrency note:** `Ring` is NOT concurrent-safe; provide external synchronization when used across goroutines.

### CCL

CCL (Column Calculation Language) is Insyra's expression language for column calculations and statement-based transforms. The `internal/ccl` package provides compilation and evaluation helpers which are useful for building tools that analyze or test CCL expressions. Any structure that implements the `engine.Context` (an alias of `ccl.Context`) interface can be used with CCL (for example, DataTable's internal context and `MapContext` implement this interface).

```go
// type alias
type CCLNode = ccl.CCLNode

// Compilation / Evaluation helpers
func CompileExpression(expression string) (CCLNode, error)
func CompileMultiline(script string) ([]CCLNode, error)
func Evaluate(node CCLNode, ctx Context) (any, error)
func EvaluateStatement(node CCLNode, ctx Context) (*EvaluationResult, error)

// Function registration
func RegisterFunction(name string, fn func(...any) (any, error))
func RegisterAggregateFunction(name string, fn func(...[]any) (any, error))
func RegisterStandardFunctions()

// MapContext for quick testing
func NewMapContext(data map[string][]any) (*MapContext, error)
```

Key notes (see `internal/ccl` and `Docs/CCL.md` for full details):

- `CompileExpression` / `CompileMultiline` compile CCL text into AST nodes (`CCLNode`).
- `Evaluate` evaluates an expression node for the current row in a `ccl.Context`.
- `EvaluateStatement` returns an `EvaluationResult` (assignment / new column metadata) but does **not** apply changes to higher-level data structures — DataTable applies assignments at a higher level.
- Call `ccl.RegisterStandardFunctions()` (from the `engine/ccl` subpackage) once to register built-in scalar and aggregate functions (e.g., `IF`, `SUM`, `AVG`, `CONCAT`). Registration is package-global (stored in `internal/ccl`'s function maps), so once registered all implementations of the `ccl.Context` interface can use these functions. It is recommended to call this at startup (e.g., in `main` or `init`) and protect with `sync.Once` if there is any chance of concurrent registration.
- `MapContext` (see `internal/ccl/map_context.go`) implements `Context` for a `map[string][]any` and is useful for tests and quick experiments.

### Sorting & Comparison Utilities

```go
func GetTypeSortingRank(v any) int
func CompareAny(a, b any) int
func ParallelSortStableFunc[S ~[]E, E any](x S, cmp func(E, E) int)
```

- `GetTypeSortingRank` returns an integer rank for type-based ordering used for mixed-type sorting (e.g., `nil < bool < number < string < time < other`). Lower ranks come first.
- `CompareAny` compares two arbitrary values using type rank and type-specific logic and returns -1, 0, or 1.
- `ParallelSortStableFunc` sorts a slice stably and in parallel when beneficial. It falls back to a single-threaded stable sort for small slices (default threshold ~4910) and uses chunked parallel sorting + stable merge for large slices.

Sorting example:

```go
// sort []int
ints := []int{5, 3, 1, 4, 2}
ParallelSortStableFunc(ints, func(a, b int) int {
    if a < b { return -1 }
    if a > b { return 1 }
    return 0
})

// sort []any (mixed types) using CompareAny
vals := []any{3, "abc", nil, 1.2, true}
ParallelSortStableFunc(vals, func(a, b any) int { return CompareAny(a, b) })
```

## Examples

- Create and use `BiIndex`:

```go
b := NewBiIndex(0)
if id, ok := b.Assign("col_A"); ok {
    fmt.Println("assigned id", id)
}
if id, ok := b.Index("col_A"); ok {
    fmt.Println("index of col_A =", id)
}
```

- Use a `Ring`:

```go
r := NewRing[int](8)
r.Push(1)
r.Push(2)
fmt.Println(r.Len()) // 2
if v, ok := r.PopFront(); ok { fmt.Println(v) } // 1
fmt.Println(r.ToSlice()) // [2]
```

- Use CCL to evaluate an expression per-row:

> Note: Any structure that implements the `ccl.Context` interface (in the `engine/ccl` subpackage) can be used to evaluate CCL expressions; `NewMapContext` is a convenient implementation for tests and quick experiments.

```go
import "github.com/HazelnutParadise/insyra/engine/ccl"

data := map[string][]any{
    "A": {1.0, 2.0},
    "B": {3.0, 4.0},
}
ctx, _ := ccl.NewMapContext(data)
node, _ := ccl.CompileExpression("A + B")
for i := 0; i < ctx.GetRowCount(); i++ {
    ctx.SetRowIndex(i)
    v, _ := ccl.Evaluate(node, ctx)
    fmt.Println(v) // 4, 6
}
```

- Register standard CCL functions for evaluation (call once via `engine`):

```go
engine.RegisterStandardFunctions()
```

- Compare two values with `CompareAny`: 


```go
cmp := CompareAny(10, "10") // type-ranking first; numeric values are ranked differently than strings
```

- Use stable parallel sort on large datasets:

```go
records := make([]Record, 100000)
// populate records...
ParallelSortStableFunc(records, func(a, b Record) int {
    // return -1, 0, 1 according to comparison
})
```

## Notes ⚠️

- See `internal/algorithms/sort.go` for the exact behavior and trade-offs of `CompareAny` and `GetTypeSortingRank`.
- `ParallelSortStableFunc` offers benefits on large slices; for small slices it falls back to a single-threaded stable sort.

## Related Links 🔗

- Internal implementation: `internal/core` (`BiIndex`)
- Algorithms implementation: `internal/algorithms` (sorting / comparison)
- Related usage examples: `Docs/DataTable.md`
