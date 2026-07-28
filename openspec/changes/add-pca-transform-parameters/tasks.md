# Tasks

## 1. Return what a projection needs
- [ ] 1.1 Add the per-column centring the fit applied to `PCAResult`
- [ ] 1.2 Add the per-column scaling, present only when the fit standardised
- [ ] 1.3 Add the training scores the fit already computes internally
- [ ] 1.4 Populate all three without changing how the decomposition itself is computed

## 2. Verify
- [ ] 2.1 Test that centring, scaling and loadings applied to new data match R's `predict(prcomp, newdata)`
- [ ] 2.2 Test that the returned training scores match a projection of the training data through the same parameters
- [ ] 2.3 Test that a covariance-based fit and a correlation-based fit are distinguishable from the result

## 3. Record
- [ ] 3.1 Document the new fields in `Docs/stats.md`
- [ ] 3.2 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
