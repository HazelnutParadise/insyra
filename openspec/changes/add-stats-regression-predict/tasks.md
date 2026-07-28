# Tasks

## 1. Predict
- [ ] 1.1 Add `Predict` to `LinearRegressionResult`, taking new predictor columns and returning one response per observation
- [ ] 1.2 Add `Predict` to `PolynomialRegressionResult`, expanding the powers the fit used
- [ ] 1.3 Add `Predict` to `ExponentialRegressionResult`, returning the response on its original scale
- [ ] 1.4 Add `Predict` to `LogarithmicRegressionResult`, likewise
- [ ] 1.5 Refuse a predictor count that does not match the fit, with an error naming the mismatch
- [ ] 1.6 Follow the existing `Predict` convention in the GLM family rather than inventing a second shape

## 2. Validate against R
- [ ] 2.1 Add reference scripts calling `predict()` for all seven families, following the existing `stats/testdata` pattern
- [ ] 2.2 Compare Insyra's predictions against them in the cross-language tests
- [ ] 2.3 Cover the five `Predict` methods that already existed — they have never been checked
- [ ] 2.4 Record what R returns that Insyra does not, as a stated gap rather than a silent difference

## 3. Record
- [ ] 3.1 Document the new methods in `Docs/stats.md`
- [ ] 3.2 Update the Go API skill so agents know regressions can predict
- [ ] 3.3 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
