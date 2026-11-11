# 0005 — `insyra/ml` imitates scikit-learn, with fitting returning a separate value

**Status:** accepted
**Date:** 2026-07-29

## Context
The library already imitated scikit-learn in four places without saying so. `datatable_scale.go` ships `Scaler` with scikit-learn's four verbs and its four scaler names, and `stats/knn.go` reproduces `KNeighborsClassifier`'s options.

## Decision
`ml` follows scikit-learn's estimator model. Fitting returns a fitted value separate from the configuration, and a pipeline step is a fit function, not a configured object.

## Options rejected
- scikit-learn's own machinery. It discovers parameters by reflecting over `__init__` and clones an unfitted estimator per fold through `inspect.signature`. Go has no equivalent short of struct-tag reflection, which this repository uses nowhere.

## Consequences
Consistency with the library's existing surface outranks other precedent. A closure refits by being called again, so cross-validation, pipelines and grid search refit by calling it, and their tests fake a whole model through `ml.Estimator.Fit`.
