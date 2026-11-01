# Tasks

- [x] Reference values: `stats/testdata/autocorrelation_reference.R` (R `Box.test`, `QRobust` from its definition) and `autocorrelation_reference.py` (statsmodels `acorr_ljungbox`), with their output.
- [x] Tests first in `stats/autocorrelation_test.go`: `Q` and `PQ` against R and statsmodels, `QRobust` and `PQRobust` against R, `QRobust` equal to the sum of `TRobust^2`, the shown header, scale invariance.
- [x] `Q`, `PQ`, `QRobust`, `PQRobust` in `stats/autocorrelation.go`, through `chiSquaredPValue`.
- [x] `Docs/stats.md`: the `### Autocorrelation` section describes the four fields, the Box-Pierce weighting of `QRobust` and that the p-values assume an observed series.
- [x] Both CHANGELOGs: the unreleased `Autocorrelation` entry under `### `stats`` names the new fields.
- [x] `openspec validate autocorrelation-portmanteau --strict` passes.
