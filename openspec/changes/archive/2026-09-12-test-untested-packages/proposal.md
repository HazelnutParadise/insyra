# Proposal: test-untested-packages

## Why

TS-11 (#307) and TS-13/TS-18 (#309): sixteen packages had no test file at all, eight sat below 50%, and `gplot` had no test for any chart it can draw.

Some of the review's numbers had already moved by the time this was picked up — `cli` (root), `internal/csv`, `internal/algorithms` and `gplot` all gained tests during the review's own batches, and `gplot` is no longer at 14.9%. The measurements below are from 2026-09-12, not from the review.

The packages this leaves are not obscure corners. `plot` is the charting API the README points at, `isr` is the entry point the README recommends for new code, `internal/utils` decides what every `Show()` prints and what every numeric read returns for a value it cannot read, and `engine/*` is the only way code outside the module can reach the actor, the ring, the index and the CCL compiler.

## What Changes

Tests only. No library or CLI behaviour changes, so no changelog entry.

| Package | Before | After | What is now covered |
| --- | --- | --- | --- |
| `gplot` | 24.8% | 79.5% | every `CreateXxxChart` rendered to a file, five output formats, the documented nil returns |
| `plot` | 0% | 72.6% | all thirteen constructors rendered to HTML, every documented nil return, `SaveHTML` including title escaping |
| `plot/internal` | 0% | 51.8% | the colour palette's wrap-around, `ApplyYAxis`'s category detection and its written-back labels |
| `isr` | 44.2% | 85.7% | the whole `DL` surface, `DT.From`'s untested cases, every `Col`/`Row`/`At` selector, the five untested window wrappers, `Name()` keys, `CCL` |
| `internal/utils` | 34.6% | 91.1% | `ToFloat64`/`ToFloat64Safe`, `TruncateString`, every `FormatValue` branch, the four timestamp windows and their boundaries |
| `lpgen` | 0% | 83.9% | model building, the generated LP file section by section, the LINGO parser, `lingoIsBound`'s four documented cases |
| `engine/ccl` | 0% | 87.5% | compile → bind → evaluate from a consumer's package, `errors.As` on the re-exported error types, the statement helpers, a custom function |
| `engine/atomic` | 0% | 100% | the documented "any struct with an Actor" pattern, serialisation, `AtomicDoN` from both lock orders |
| `engine/ring`, `engine/biindex`, `engine/algorithms` | 0% | 100% | the forwarders, and the aliases spelled out where the compiler checks them |
| `stats/internal/parutil` | 0% | 97.1% | `ChunkBounds` tiles `[0, n)` exactly, and agrees with the partition `Run` uses |
| `datafetch/internal/limiter` | 0% | 100% | spacing, concurrent callers, the cancellation rollback, and the invariant that two allowed calls are never closer than the interval |
| `cli/style` | 0% | 100% | the colour switch in both positions |

Cases that would hang rather than fail — a duplicate actor passed to `AtomicDoN`, two goroutines locking a pair in opposite orders — run behind a one-second watchdog.

One doc comment is corrected: `engine/ccl.EvalError` said to match it with `errors.As`, which never works on what `Evaluate` returns. The comment now says the DataTable methods are what wrap a failure in one, and a test pins it.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `test-suite-integrity`: gains the rule that a package's exported surface is tested in that package, not only indirectly through a caller, and that a chart is tested by rendering it.

## Impact

- Twelve new test files. One doc comment in `engine/ccl/ccl.go`. No other non-test change.
- `api-review.md` rows TS-11, TS-13 and TS-18; `delivery-status.md`.
- **Deliberately left untested, with reasons**: `allpkgs` and `engine` (blank imports and a doc comment — the compiler already covers them), `cmd/insyra` (sixteen lines that call `cli.Execute`), `py` (needs the embedded Python environment; its `py/internal/ipc` is already at 80%), `tools/gendocs` (a build tool that rewrites `Docs/` in place).
- **TS-18 is not finished.** Three of its eight packages are done. `cli/repl` (39.1%), `datafetch` (44.5%), `ml/mltest` (40.7%), `stats/internal/fa` (22.2%) and `accel/knnbridge` (21.1%) need a pass of their own: the first two need an interactive session and HTTP fixtures, and the last is low because its tests are GPU-gated, which is #303's subject rather than this one's.
- Writing these found five defects, fixed in their own changes rather than here: five panics in `gplot` and two in `plot` on ordinary bad input, a slice-bounds panic in the LINGO parser, `isr.DT.From(map[int]any{…})` never producing a table, `utils.FormatValue` answering differently on amd64 and arm64 above 2^63, and `utils.TruncateString` panicking on a negative width.
