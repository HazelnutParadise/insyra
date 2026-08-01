# Tasks

## 1. Implementation
- [x] 1.1 `ExactSplits` on `DecisionTreeOptions`; refuse an explicit `MaxBins` alongside it
- [x] 1.2 Exact numeric encoding: one boundary per adjacent distinct-value pair, feeding the existing boundary scan
- [x] 1.3 Nothing else changes: categorical, missing routing, bounds, importances

## 2. Verification
- [x] 2.1 Prediction-for-prediction against scikit-learn: classification labels exact at depth 4; regression at depth 3 with a 5-sample leaf floor, within single-precision tolerance. The regression bound is not a hedge: deeper trees reach nodes of two or three rows where several splits remove all variance and tie exactly, and scikit-learn breaks those ties with a per-node random feature order no deterministic implementation can match — measured, root split identical at every depth, divergence only in tie territory
- [x] 2.2 The refusal
- [x] 2.3 Exact and histogram trees differ on data where the quantile edges miss the true boundary — the case that shows the option does something
- [x] 2.4 Ensembles accept the option through the passthrough

## 3. Documentation
- [x] 3.1 `Docs/ml.md` decision-tree section: both styles, the cost trade, the default
- [x] 3.2 `skills/insyra/`; changelogs in both languages
