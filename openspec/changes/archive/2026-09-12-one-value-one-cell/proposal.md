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

## Backport to dev (0.3.x)

Dev received all of it: `Cell`, the unexported marker and `unwrapCell`/`unwrapCells`; `flattenWithNilSupport` storing a marked value whole; `Append`, `Update`, `InsertAt`, the three `Replace` methods, `UpdateElement` and both row appenders unwrapping; `equalCell` and `valueMatcher` unwrapping, so a search with a marked value agrees with the bare one; the tests, the `Docs/DataList.md` paragraph, both changelogs, and the AGENTS.md follow-up on a bare `[]byte` flattening, re-checked here.

Adapted:
- The unwrap for the new value sits in `replaceAll_notAtomic`, `replaceFirst_notAtomic`, `replaceLast_notAtomic` and `replaceNaNsAndNilsWith_notAtomic` rather than in the public `Replace*` methods, because `DataTable.Replace` and `ReplaceInCol` call those helpers directly and would otherwise store the marker (0.4 does).
- Entry points 0.4's list missed also unwrap, each with a test: `DataList.ReplaceNilsWith`, `ReplaceNaNsWith` and `ReplaceNaNsAndNilsWith`, `Shift`'s fill value (and so `DataTable.ShiftCol`; the mark is left on there, because `Shift` builds its result with `NewDataList`, which stores a marked value whole), and `DataTable.ReplaceNaNsAndNilsWith`, `ReplaceInRow`, `ReplaceNaNsAndNilsInRow` and `ReplaceNaNsAndNilsInCol`. `FindRowsIfContainsAll` unwraps before its NaN check, so `Cell(NaN)` behaves like a bare `NaN` there.
- The docs and changelog name those entry points; the docs' "grouped and ordered by content" becomes "counted and matched", because on this line grouping keeps its existing keys.
- Values returned from a `Map` or `Filter` callback are not unwrapped, as on 0.4.

Left on 0.4:
- `delivery-status.md` edits.
