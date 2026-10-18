# Proposal: self-referential-cells-print-by-type

## Why

A slice that contains itself, put into a list with `Append`, ended the process with a fatal stack overflow (not recoverable) when `Show` printed it or when a `stats` function refused it and formatted the error with `%v` (AGENTS.md follow-up 2026-09-12).

## What Changes

- `internal/utils.SprintBounded` prints a value with `%v` only when a walk of at most 16 levels and 10,000 nodes finishes; otherwise it prints the value's type. `FormatValue` uses it for slice elements, and `stats`' non-numeric-cell errors use it for the refused cell.
- Ordinary cells print exactly as before.

## Capabilities

### Modified Capabilities

- `cell-display-and-export`: a cell that refers to itself is shown by type.

## Impact

`internal/utils/utils.go`, `stats/numericinput.go`, `stats/factor_analysis.go`, tests, both CHANGELOGs, the follow-up entry in `AGENTS.md`.
