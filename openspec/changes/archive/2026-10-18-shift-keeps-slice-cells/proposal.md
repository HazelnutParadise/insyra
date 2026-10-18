# Proposal: shift-keeps-slice-cells

## Why

`DataList.Shift` builds its result with `NewDataList`, which flattens every slice, so a cell holding a slice is split into several cells. Measured on 2026-09-14: `Append([]byte{1, 2}, 3)` then `Shift(0)` gives `[1 2 3]`, three cells where there were two.

## What Changes

- `Shift` wraps each existing cell with `Cell` before building the result, so a stored slice stays one cell and the result has the length of the input.
- The fill value keeps the rule it had: a bare slice fill flattens, a `Cell`-marked one stays whole.

## Capabilities

### Modified Capabilities

- `datalist-construction`: Shift keeps cell boundaries.

## Impact

`datalist_window.go`, `flatten_depth_test.go`, `AGENTS.md`, both CHANGELOGs.
