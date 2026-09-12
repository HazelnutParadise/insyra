# Proposal: one-value-one-cell

## Why

`NewDataList` flattens slices on purpose, so that constructing a list reads the way constructing a pandas Series does:

```go
NewDataList([]int{1, 2, 3})   // three cells
```

There is no way to opt out for one argument. A caller who wants a whole slice in one cell has to leave the constructor entirely and use `Append`, which never flattens — so they cannot mix the two in a single call, and the asymmetry is not written down anywhere:

```go
NewDataList([]int{1, 2})   // two cells
Append([]int{1, 2})        // one cell
```

Since `identify-uncomparable-cells` a slice in a cell works properly — it is counted, matched, grouped and ordered by its content — so there is now a reason to put one there deliberately.

## What Changes

- `Cell(v)` marks a value so a constructor stores it whole. The marker is unexported and is unwrapped on the way in, so the cell holds `v` at its own type, not a wrapper:

```go
NewDataList(Cell([]int{1, 2}), 3, "a")   // three cells: []int{1,2}, 3, "a"
```

- Every entry point that takes a value from the caller unwraps the marker, so the two spellings agree and a marker can never end up stored as a cell: `Append`, `Update`, `InsertAt`, `ReplaceFirst`, `ReplaceAll`, `UpdateElement`, and the two row appenders that take a map. `Append(Cell(x))` and `Append(x)` mean the same thing, because `Append` does not flatten in the first place — it is accepted so that writing it for consistency is not a trap.
- The search side unwraps too, so `Count(Cell(x))` and `Count(x)` agree.
- Bare `NewDataList([]byte{…})` still flattens. That was decided on 2026-09-12: the flattening is what makes the constructor read like a Series, and a `[]byte` is not special enough to carve out.

## Capabilities

### New Capabilities

- `cell-construction`: when a value passed to a constructor becomes one cell and when it becomes several.

### Modified Capabilities

(none)

## Impact

- `Cell` is a new exported name. Nothing existing changes: a call without it behaves exactly as before, and the marker is only produced by `Cell`.
- A marker reaching a cell would be a defect, so every value-taking entry point is covered rather than only the constructors.
