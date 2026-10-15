# Proposal: mode-returns-every-mode

## Why

[#221](https://github.com/HazelnutParadise/insyra/issues/221) (api-review D-13) found two problems with `DataList.Mode()`, measured on `0.4` at 6493910b:
- **It returned nil whenever every value occurred equally often.** That included `[1, 2, 3]`, `[1, 1, 2, 2]` and a one-value list `[7]`, whose mode is 7 under any definition.
- **Its order was not stable.** When several values tied, the order came from map iteration: `[3, 3, 1, 1, 2]` returned `[3 1]` or `[1 3]` from one call to the next.

pandas 3.0.5's `Series.mode` returns `[1, 2, 3]`, `[1, 2]`, `[7]` and `[1, 3]`. Its documentation says "Modes of the Series in sorted order". The owner ruled on 2026-10-15 to follow pandas.

## What Changes

- `Mode()` returns every value that reaches the highest count, in ascending order. A list whose values all occur equally often returns every value. `nil` and `NaN` cells are skipped, and a list with no numeric values still returns nil.
- The return type stays `[]float64`, so no caller stops compiling. A caller that read `nil` as "no mode" now gets values, which is marked BREAKING in the changelog.
- `Docs/DataList.md` describes the new rule. The `mode` CLI command prints the new result as is.

## Capabilities

### New Capabilities

- `datalist-mode`: what `Mode` returns.

### Modified Capabilities

None.

## Impact

- `datalist.go` `Mode`, and a new test file.
- `Summary` already prints at most a few modes. Its truncation is unchanged.
