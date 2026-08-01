# Change: Splitting, cross-validation and metrics

## Why
A model package without a way to measure a model is a demo. Everything here is unglamorous and none of it is optional: without a train/test split there is no honest number, without cross-validation there is no stable one, and without metrics there is nothing to report.

`DataTable.TrainTestSplit` already exists in the root package (`datatable_sampling.go:188`), so the splitting half is largely wiring. The rest is small: each metric is a loop, and cross-validation is a loop over folds calling a fit function that already exists in the protocol.

Cross-validation is also what makes the pipeline's value visible. Refitting a pipeline per fold refits its preprocessing per fold, which is the only arrangement where a fold's score means anything.

## What Changes
- Add k-fold splitting, including a stratified form that preserves class balance
- Add cross-validation over an estimator, refitting per fold
- Add the metrics a caller needs to report a result: accuracy, log loss, ROC AUC, confusion matrix for classification; RMSE, MAE and R² for regression
- Take the metric as an argument rather than defaulting it by model family, so a reported score always names what it measured

## Impact
- Affected specs: `ml-model-selection`
- Affected code: `ml/`
- Depends on `add-ml-estimator-protocol`
