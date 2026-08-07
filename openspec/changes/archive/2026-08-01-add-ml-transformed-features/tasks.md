# Tasks

## 1. Capture
- [x] 1.1 Record the fully transformed table's column names when a pipeline is fitted
- [x] 1.2 Fall back to the estimator's own feature names when a pipeline has no steps

## 2. Expose
- [x] 2.1 Add the optional capability interface, following the shape the package's other optional capabilities use
- [x] 2.2 Implement it on every fitted-pipeline variant, so the capability does not disappear when a pipeline also reports probabilities or importances

## 3. Conformance
- [x] 3.1 Require, in the conformance helper, that a model's importance count matches its feature-name count — against the transformed names where the model reports them
- [x] 3.2 Verify the new check fails against a model that violates it and passes against one that does not

## 4. Leakage
- [x] 4.1 Test that a pipeline cross-validated with a fitted-parameter step transforms a held-out row using the training-fitted step, distinguishable from a step fitted on the whole dataset

## 5. Documentation
- [x] 5.1 Document the capability in `Docs/ml.md` and state that importances are indexed by the transformed names
- [x] 5.2 Update `skills/insyra/`
- [x] 5.3 Add the entry to `CHANGELOG.md` and `CHANGELOG_TW.md`
