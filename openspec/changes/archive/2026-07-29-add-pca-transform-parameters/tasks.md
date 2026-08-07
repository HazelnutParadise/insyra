# Tasks

## 1. Return what a projection needs
- [x] 1.1 Add the per-column centring the fit applied to `PCAResult`
- [x] 1.2 Add the per-column scaling, present only when the fit standardised
- [x] 1.3 Add the training scores the fit already computes internally
- [x] 1.4 Populate all three without changing how the decomposition itself is computed

## 2. Verify
- [x] 2.1 Test that centring, scaling and loadings applied to new data match R's `predict(prcomp, newdata)`
- [x] 2.2 Test that the returned training scores match a projection of the training data through the same parameters
- [x] 2.3 Test that a covariance-based fit and a correlation-based fit are distinguishable from the result

## 3. Record
- [x] 3.1 Document the new fields in `Docs/stats.md`
- [x] 3.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
