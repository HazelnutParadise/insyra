# Proposal: stats-nil-lists-are-errors

## Why

A nil `*DataList`, or a wrapper holding one such as `isr.DL`, reached `AtomicDo` in the regression, correlation, chi-square, silhouette, KNN, skewness and kurtosis entry points and panicked. In the regressions the panic happened in a goroutine the function starts itself, which no caller can recover (AGENTS.md follow-up 2026-09-14).

## What Changes

- A nil list is an error everywhere in `stats` that reads one, as an empty list already is for the hypothesis tests. `insyra.IsNilList` is the test, so wrappers count.
- `insyra.ProcessData` treats a nil list like an unsupported input: a warning and `nil, 0`, so `Skewness` and `Kurtosis` return their empty-data error.

## Capabilities

### Modified Capabilities

- `stats-input-type-guard`: a nil list is refused with an error.

## Impact

`stats/numericinput.go`, `regression_shared.go`, `regression_logistic.go`, `chi_square.go`, `clustering.go`, `knn.go`, `utils.go`, tests, both CHANGELOGs, the follow-up entry in `AGENTS.md`.
