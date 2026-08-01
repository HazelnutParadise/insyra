# Change: Random forests over the trees the package already grows

## Why
A single decision tree is high-variance: small changes in the training data move its splits, and its accuracy ceiling on tabular data is well below what the same tree averaged over resamples reaches. The random forest — bootstrap the rows, restrict each split to a random feature subset, average the trees — is the standard remedy and the most-reached-for tabular model in applied work. The trees exist; the forest is the missing assembly.

The tree fitting was refactored for this rather than duplicated: preparation (feature encoding, class collection, target quantisation) is computed once from the full data, and each tree grows from a bootstrap row multiset — a repeated index simply counts twice in every statistic, which is exactly what resampling with replacement means. Because classes come from the full target, every tree shares one class order even when its bootstrap sample misses a class, which removes the column-alignment failure a per-tree class list would invite.

## What Changes
- Add `FitRandomForestClassifier` and `FitRandomForestRegressor` with scikit-learn's defaults: 100 trees, √p features per split for classification and all p for regression, probability averaging rather than majority voting
- Add per-split feature subsampling to the tree growth, off by default so single trees are unchanged
- Fit trees in parallel with per-tree RNGs derived from one forest seed, so the result is identical however goroutines interleave; an unseeded fit draws a seed and reports it on the model
- Aggregate importances as the renormalized mean of the per-tree normalized importances
- Verify: seeded determinism, probability validity, signal-over-noise importance ranking, conformance, and accuracy agreement with scikit-learn on separable data

## Impact
- Affected specs: `ml-trees`
- Affected code: `ml/random_forest.go`, `ml/decision_tree.go` (preparation split out, feature subsampling), docs, changelogs, `skills/insyra/`
- Additive; single-tree behaviour is unchanged and its tests still pass
