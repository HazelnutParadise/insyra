# Tasks

## 1. Splitting
- [ ] 1.1 Add k-fold splitting over a table, every observation in exactly one fold
- [ ] 1.2 Add a stratified form preserving class proportions
- [ ] 1.3 Make splits reproducible from a seed
- [ ] 1.4 Reuse `DataTable.TrainTestSplit` rather than writing a second splitter
- [ ] 1.5 Error when a class is too small to stratify, naming it

## 2. Cross-validation
- [ ] 2.1 Add cross-validation over an `Estimator`, refitting per fold
- [ ] 2.2 Return a score per fold, not only the mean
- [ ] 2.3 Report the fold a failure occurred on

## 3. Metrics
- [ ] 3.1 Classification: accuracy, log loss, ROC AUC, confusion matrix
- [ ] 3.2 Regression: RMSE, MAE, R²
- [ ] 3.3 Take the metric as an argument; carry its name on the result
- [ ] 3.4 Refuse a metric that does not apply to the model

## 4. Verify
- [ ] 4.1 Test that folds partition the data with nothing lost or duplicated
- [ ] 4.2 Test that a stratified split preserves class proportions within a stated tolerance
- [ ] 4.3 Test that cross-validating a pipeline refits its preprocessing per fold — construct a case where not doing so would leak and show it does not
- [ ] 4.4 Check each metric against a worked example computed independently
- [ ] 4.5 Test that a mismatched metric is refused

## 5. Record
- [ ] 5.1 Document splitting, cross-validation and metrics in `Docs/ml.md`
- [ ] 5.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
