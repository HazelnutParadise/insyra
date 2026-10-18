# Proposal: pca-knn-refuse-non-finite

## Why

`PCA` and the targets `KNNRegress` reads check only that a cell converts to a number, so `NaN` and `±Inf` reach the result with a nil error: `PCA` returns `NaN` eigenvalues and `KNNRegress` predicts `NaN` or `±Inf`. Every other numeric entry point of `stats` already refuses such a value, and `Docs/stats.md` lists these two as the exceptions. The owner asked on 2026-10-18 for bugs to be fixed in 0.3.5.

## What Changes

- `PCA` returns an error when any cell is `NaN` or `±Inf`.
- `KNNRegress` returns an error when any target is `NaN` or `±Inf`.
- A call that already returned an error is unchanged; one that returned a `NaN` or infinite result now returns an error.

## Capabilities

### Modified Capabilities

- `stats-input-type-guard`: PCA and KNNRegress targets refuse non-finite values.

## Impact

`stats/pca.go`, `stats/knn.go`, `stats/pca_knn_nonfinite_test.go`, `Docs/stats.md`, `AGENTS.md`, both CHANGELOGs.
