# Proposal: newdatalist-bounds-slice-depth

## Why

`flattenWithNilSupport` recurses into every slice with no limit. A slice that contains itself exhausts the stack, which is a `fatal error`, not a panic, so no caller can recover and the whole program ends. Measured on 2026-09-13: `cyclic := []any{1}; cyclic[0] = cyclic; insyra.NewDataList(cyclic)`. Recorded in `AGENTS.md` since 2026-09-12.

## What Changes

- The flattener descends at most 64 slices deep; a slice found below that is kept as one cell, the same thing `Cell` asks for.
- Nesting up to 64 deep flattens exactly as before. Cost on the constructor's path is one integer comparison per slice.
- The `AGENTS.md` follow-up is narrowed to what remains: `Append` can still hold such a slice, and printing it overflows the stack.

## Capabilities

### New Capabilities

- `datalist-construction`: how NewDataList treats nested input.

## Impact

`datalist.go`, `flatten_depth_test.go`, `AGENTS.md`, both CHANGELOGs.
