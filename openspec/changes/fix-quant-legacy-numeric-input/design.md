# Design: fix-quant-legacy-numeric-input

## Context

`quant/bootstrap.go` defines `numericSeries(dl insyra.IDataList, label string) ([]float64, error)`: nil check, `AtomicDo` read, `insyra.ToFloat64Safe` per cell, refusal of NaN/Inf with a one-based row. Every function added since uses it. `quant/performance.go` wraps its `...F64` cores with `returns.ToF64Slice()` and friends; `quant/overfitting.go` does the same for `DeflatedSharpeRatio` and inside `PBO`'s column loop. `ToF64Slice` returns a full-length slice with `0` for anything unparsable, so the cores never see a problem. The AGENTS.md follow-up dated 2026-08-01 lists these among 54 callers deliberately left alone because they were "display and reporting paths"; that is no longer true of `quant`, which feeds VaR, Calmar, and factor attribution from the same series.

## Goals / Non-Goals

**Goals:**
- One input contract across `quant`: unreadable cells are errors with a location.
- Zero change to numeric results on clean input.

**Non-Goals:**
- Changing `ToF64Slice` or its remaining callers outside `quant`.
- Any new validation beyond readability (no minimum-sample changes, no sign checks).
- Touching `WalkForward`, whose inputs are already `[]float64`.

## Decisions

### Replace the wrapper line only

Each exported function swaps `x.ToF64Slice()` for `values, err := numericSeries(x, "<label>")` and returns the error; the `...F64` core is untouched, so every existing numeric test keeps passing without edits. `PBO` labels each column `column <j>` with the zero-based `j` its existing length-mismatch message already uses, so the two messages agree.

### Nil becomes an error, not a panic

`ToF64Slice` on a nil interface panics today; `numericSeries` returns "`<label>` is nil". This is a strict improvement and is specified so it cannot regress.

### Marked breaking, no compatibility flag

A caller who relied on blanks becoming zeros was getting a wrong number. Offering a "lenient" option would preserve the wrong number behind a flag; the fix for a series with gaps is `ClearNils` or `FillNaN…` before the call, which the docs already show. The changelog entry carries the breaking marker used by past release notes.

## Risks / Trade-offs

- [Downstream code passing series with leading `nil` from `PctChange`] → it now errors instead of computing a slightly wrong Sharpe; the error names the row and the docs point at `ClearNils`. This is the intended outcome.
- [Follow-up entry drifts] → the AGENTS.md follow-up is edited in the same change to drop the `quant` callers, keeping the remaining list accurate.

## Open Questions

None.
