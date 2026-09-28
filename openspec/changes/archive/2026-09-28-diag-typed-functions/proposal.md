# Proposal: diag-typed-functions

## Why

`stats.Diag(x any, dims ...int) (any, error)` imitates R's `diag()`: given a matrix it returns the diagonal, given a slice a diagonal matrix, given a number an identity matrix, and `dims` resizes the result (#245, ST-9 in `api-review.md`). Because it takes and returns `any`, every caller type-asserts the result, a wrong argument type is found only at run time, and the documentation has to explain four functions under one name. It also panics, which library code must never do: `Diag(0)`, `Diag([]float64{})` and `Diag(-1)` ask gonum for a zero- or negative-sized matrix, and gonum panics.

## What Changes

- Four typed functions replace it, one per job:
  - `DiagOf(m mat.Matrix) ([]float64, error)` returns the main diagonal of any gonum matrix (`Diag` took only `*mat.Dense`).
  - `DiagMatrix(v []float64) (*mat.Dense, error)` returns the `len(v)`×`len(v)` matrix with `v` on its diagonal.
  - `DiagMatrixSize(v []float64, nrow, ncol int) (*mat.Dense, error)` returns an `nrow`×`ncol` matrix with `v` on its diagonal and zeros past the end of `v`, the case `Diag(v, nrow, ncol)` covered, including the rectangular identity (`Diag(nil, 2, 3)` is `DiagMatrixSize([]float64{1, 1}, 2, 3)`). A `v` longer than the diagonal is an error instead of being cut silently.
  - `IdentityMatrix(n int) (*mat.Dense, error)` returns the `n`×`n` identity.
  An empty `v`, a size below 1, and a nil matrix are errors.
- `Diag` stays for one release, **Deprecated**, with its old meaning, under the one-name rule; its doc comment maps each form to its replacement. The sizes that made it panic now return an error. Its removal is an `AGENTS.md` follow-up.
- `Docs/stats.md`, both CHANGELOGs, `api-review.md` (the `Diag` part of ST-9), `delivery-status.md` and `AGENTS.md` are updated.

## Capabilities

### New Capabilities

- `stats-matrix`: the typed diagonal and identity matrix helpers in `stats`.

### Modified Capabilities

(none)

## Impact

- Code: `stats/diag.go`. Nothing else in the module calls `Diag`.
- Tests: `stats/diag_test.go`.
- Docs: `Docs/stats.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`, `AGENTS.md`. The skills do not mention `Diag`.
- No new dependencies.
