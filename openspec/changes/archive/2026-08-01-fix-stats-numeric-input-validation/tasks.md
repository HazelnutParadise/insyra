# Tasks

## 1. The shared converter
- [x] 1.1 Add a validating converter to `stats` that turns a `DataList`'s values into `[]float64`, refusing anything that is not a finite number and naming the offending row
- [x] 1.2 Route `numericMatrixFromTable` through it so clustering, PCA and KNN keep their current behaviour from one implementation rather than two
- [x] 1.3 Give the converter a test that pins each refusal: missing value, text, infinity, undefined value

## 2. Regression
- [x] 2.1 Route `gatherRegressionInputs` through the converter for target, predictors and offsets
- [x] 2.2 Route the polynomial-regression input reads through it
- [x] 2.3 Route the GLM prediction offset through it
- [x] 2.4 Test that each affected family refuses a missing predictor and a missing target, and that the error names the row

## 3. Correlation
- [x] 3.1 Route every correlation input read through the converter
- [x] 3.2 Test that a correlation over a series holding a blank is refused rather than scored — the case measured at r=0.9879 against a true 0.9992

## 4. Confirm the untouched families still behave
- [x] 4.1 Test that the decision tree still accepts a missing feature value and still refuses a missing target — its policy is deliberate and must not be swept into the refusal rule
- [x] 4.2 Test that factor analysis still applies listwise deletion

## 5. Documentation
- [x] 5.1 State each family's treatment of unreadable values in `Docs/stats.md`
- [x] 5.2 Mirror it in `Docs/ml.md` for the wrapped families
- [x] 5.3 Add the breaking entries to `CHANGELOG.md` and `CHANGELOG_TW.md`
