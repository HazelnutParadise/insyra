# Tasks

## 1. Pipeline
- [x] 1.1 Add a pipeline of `Step` values ending in an `Estimator`, fitting in order and feeding each step's output to the next
- [x] 1.2 Make the fitted pipeline satisfy `Model`
- [x] 1.3 Name the failing step in any fitting error
- [x] 1.4 Refit cleanly — fitting the same definition twice must derive nothing from the first fit

## 2. Column scoping
- [x] 2.1 Add a way to apply a transformer to named columns only, passing the rest through unchanged
- [x] 2.2 Error on a named column the data does not contain
- [x] 2.3 Preserve column order through the scoped transform

## 3. Verify
- [x] 3.1 Test that a fitted pipeline equals applying its steps and model by hand, in order
- [x] 3.2 Test that fitting twice on different data gives parameters derived only from the second
- [x] 3.3 Test the failing-step error names the step
- [x] 3.4 Test a pipeline mixing a scaler on numeric columns with an encoder on categorical ones
- [x] 3.5 Test that the root package's scalers and encoders work as steps with no adapter

## 4. Record
- [x] 4.1 Document pipelines in `Docs/ml.md`, including why fitting preprocessing before splitting leaks
- [x] 4.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`

## 5. Review remediation
- [x] 5.1 Preserve classifier, probability, and feature-importance capabilities on fitted pipelines
- [x] 5.2 Canonicalize raw input columns before applying pipeline steps, including position-sensitive external steps
- [x] 5.3 Preserve pass-through columns by index in `ColumnTransformer`, including unnamed columns
