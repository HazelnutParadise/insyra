## Why

`ToFloat64`, `ToFloat64Safe` and `ReadSlice2D` in the root package, and `CAI` in `mkt`, are exported as variables that hold a function (K-12 of #211). Anyone can reassign one, and because the library itself calls `ToFloat64Safe` from more than forty places in `stats`, `ml`, `nn`, `quant` and the core, a single `insyra.ToFloat64Safe = …` changes every t-test, ANOVA and decision tree in the process. Go's documentation lists them under Variables, where someone looking for a function does not look.

`ReadSlice2D` and `Slice2DToDataTable` are also one function under two names. The owner ruled on 2026-09-25 to keep `ReadSlice2D`, which sits with `ReadCSV`, `ReadJSON`, `ReadExcel` and `ReadSQL`, so typing `insyra.Read` lists every way to get a table.

## What Changes

- `ToFloat64`, `ToFloat64Safe`, `ReadSlice2D` and `mkt.CAI` become function declarations. Calls are unchanged; only assigning to them stops compiling. **BREAKING** for that assignment alone.
- `ReadSlice2D` holds the implementation. `Slice2DToDataTable` stays for one release as a Deprecated wrapper and is then removed, the same schedule `SetDontPanic` follows.
- A test parses the module, without type-checking it, and fails on any exported package-level variable whose value is a function literal or a function declared in this module, so the shape cannot come back unnoticed.
- The module's own callers (`isr`, `py`) and the documentation move to `ReadSlice2D`.

## Capabilities

### New Capabilities
- `exported-functions`: an exported function is declared as a function, and each has one name.

### Modified Capabilities
None.

## Impact

- `utils.go`, `read.go`, `mkt/cai.go`, `isr/dt.go`, `py/pyresult_decode.go`; a new root test.
- `Docs/DataTable.md`, `Docs/mkt.md`; both changelogs; `skills/insyra/` if it names either spelling.
- `api-review.md` K-12, `delivery-status.md`, `AGENTS.md` follow-up to remove `Slice2DToDataTable`, issue #211.
