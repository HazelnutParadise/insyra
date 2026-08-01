# Change: Exact-split trees — the scikit-learn style alongside the histogram style

## Why
The trees split on histogram bins: numeric features are quantised to at most MaxBins quantile edges, and splits are chosen among those edges. That is LightGBM's design — fast, memory-light, and the right default — but it is an approximation of the split search, and it is the one structural reason the tree family cannot be verified number-for-number against scikit-learn, whose CART evaluates every midpoint between adjacent distinct values. Both styles are standard; a library claiming both parentages should offer both.

The implementation is smaller than it sounds, because exact splitting is the histogram algorithm at its limit: put one bin boundary at the midpoint of every pair of adjacent distinct values and the existing scan over bin boundaries evaluates exactly the candidate set CART does. The split criteria already match scikit-learn's defaults — Gini for classification, variance for regression — so with the same candidate set, the same tree comes out, and predictions can be compared one for one rather than at accuracy level.

## What Changes
- Add `ExactSplits` to `DecisionTreeOptions`: numeric features get one boundary per adjacent distinct-value pair, `MaxBins` does not apply and combining the two is refused
- Categorical handling, missing-value routing, depth and leaf bounds are unchanged — they were never approximated
- Ensembles inherit the option through their `Tree` passthrough
- Verify against scikit-learn prediction-for-prediction: a classification tree and a regression tree fitted with the same depth on the same data must predict identically on a probe grid, not merely score the same accuracy
- Document the trade: exact splits cost O(distinct values) candidates per feature per node against the histogram's O(MaxBins), which is why histogram stays the default

## Impact
- Affected specs: `ml-trees`
- Affected code: `ml/decision_tree.go`, docs, changelogs, `skills/insyra/`
- Additive; the default behaviour is unchanged
