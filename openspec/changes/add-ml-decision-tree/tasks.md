# Tasks

## 1. Binning
- [x] 1.1 Bin each numeric feature by quantile, deterministically for given data and options
- [x] 1.2 Hold binned features at single precision or narrower, per the precision contract
- [x] 1.3 Give missing values their own bin rather than substituting a value

## 2. Growing
- [x] 2.1 Build per-node histograms and choose the split by gain
- [x] 2.2 Accumulate in fixed-point integers so the sum does not depend on the order of summation
- [x] 2.3 Resolve equal gains by a stated rule, not by evaluation order
- [x] 2.4 Learn a missing-value direction per split, choosing whichever scored better
- [x] 2.5 Split categorical features by a subset of categories rather than by an imposed order
- [x] 2.6 Honour the depth, leaf-count and minimum-leaf-size bounds

## 3. Scoring
- [x] 3.1 Predict labels for classification and values for regression
- [x] 3.2 Report class probabilities matching the reported classes in order
- [x] 3.3 Route a value missing at scoring time whose split saw none at fitting time, by a stated default
- [x] 3.4 Route an unseen category by a stated default
- [x] 3.5 Report leaf values and importances in double precision

## 4. Verify
- [x] 4.1 Test that the same data and options fit an identical tree, twice, including tied splits
- [x] 4.2 Test that a permutation of the input rows fits the same tree — this is what fixed-point accumulation buys and the test that proves it
- [x] 4.3 Test the learned missing-value direction against a case where the two directions differ
- [x] 4.4 Test a categorical split against a case where an imposed numeric order gives a worse tree
- [x] 4.5 Test every growth bound is respected
- [x] 4.6 Compare accuracy against scikit-learn on a shared dataset, and record the difference rather than asserting equality — the algorithms differ and pretending otherwise would be false

## 5. Record
- [x] 5.1 Document the tree in `Docs/ml.md`, including what the missing-value and unseen-category defaults are
- [x] 5.2 Record in `delivery-plan.md` that split finding is device-eligible and unmeasured
- [x] 5.3 Changelog entry in `CHANGELOG.md` and `CHANGELOG_TW.md`
