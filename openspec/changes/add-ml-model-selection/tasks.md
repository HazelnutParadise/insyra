# Tasks

## 1. Splitting
- [x] 1.1 Add k-fold splitting over a table, every observation in exactly one fold
- [x] 1.2 Add a stratified form preserving class proportions
- [x] 1.3 Make splits reproducible from a seed
- [x] 1.4 Reuse `DataTable.TrainTestSplit` rather than writing a second splitter
- [x] 1.5 Error when a class is too small to stratify, naming it

## 2. Cross-validation
- [x] 2.1 Add cross-validation over an `Estimator`, refitting per fold
- [x] 2.2 Return a score per fold, not only the mean
- [x] 2.3 Report the fold a failure occurred on

## 3. Metrics
- [x] 3.1 Classification: accuracy, log loss, ROC AUC, confusion matrix
- [x] 3.2 Regression: RMSE, MAE, R²
- [x] 3.3 Take the metric as an argument; carry its name on the result
- [x] 3.4 Refuse a metric that does not apply to the model

## 4. Verify
- [x] 4.1 Test that folds partition the data with nothing lost or duplicated
- [x] 4.2 Test that a stratified split preserves class proportions within a stated tolerance
- [x] 4.3 Test that cross-validating a pipeline refits its preprocessing per fold — construct a case where not doing so would leak and show it does not
- [x] 4.4 Check each metric against a worked example computed independently
- [x] 4.5 Test that a mismatched metric is refused

## 5. Record
- [x] 5.1 Document splitting, cross-validation and metrics in `Docs/ml.md`
- [x] 5.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
