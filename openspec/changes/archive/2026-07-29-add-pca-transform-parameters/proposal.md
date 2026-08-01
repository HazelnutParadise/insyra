# Change: Give PCA the parameters needed to project new data

## Why
`PCAResult` carries the loadings, the eigenvalues and the explained variance (`stats/pca.go:16-20`) — everything needed to describe a fitted decomposition, and not quite enough to use one.

Two things are missing. The centring and scaling the fit applied are not returned, so new data cannot be put on the same footing as the training data before projecting it. And the training data's own scores are not returned either, so a caller who wants the transformed table has to recompute what the fit already produced internally.

Both matter for the same reason: PCA is a transformer, and a transformer without the ability to transform is a description of one. R's `prcomp` returns `center`, `scale` and `x` for exactly this reason, and `predict(prcomp, newdata)` is built on them.

It also blocks two things concretely. A PCA step cannot appear in a pipeline that will later be applied to unseen data. And PCA cannot be serialised — the ONNX form of a principal-component transform is a subtraction followed by a matrix multiply, and the subtraction needs the means.

## What Changes
- Return the per-column centring the fit applied
- Return the per-column scaling, so a correlation-based fit can be distinguished from a covariance-based one and reproduced
- Return the training scores the fit already computed
- Name them after what they are rather than after `prcomp`, but verify against `prcomp` since that is the reference the rest of `stats` is checked against

## Impact
- Affected specs: `stats-decomposition`
- Affected code: `stats/pca.go`
- Additive: three new fields on a result struct, no signature changes
