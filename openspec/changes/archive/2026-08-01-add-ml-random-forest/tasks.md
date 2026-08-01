# Tasks

## 1. Tree groundwork
- [x] 1.1 Split tree fitting into reusable preparation and growth on a row multiset
- [x] 1.2 Per-split feature subsampling on the growth, sorted so tie-breaking matches the exhaustive path, off by default
- [x] 1.3 Existing single-tree tests still pass

## 2. The forest
- [x] 2.1 Options with scikit-learn's defaults; classifier and regressor fit functions
- [x] 2.2 Parallel tree fitting with per-tree RNGs derived from the forest seed
- [x] 2.3 Probability-averaged classification, mean regression, aggregated importances
- [x] 2.4 Tree options routed through the single-tree defaulting — found by a zero MaxBins panicking the histogram builder

## 3. Verification
- [x] 3.1 Conformance for both; probabilities valid; informative features outrank noise
- [x] 3.2 Same seed twice → identical; reported seed reproduces an unseeded fit
- [x] 3.3 Accuracy agreement with scikit-learn on separable data, behind the reference gate
- [x] 3.4 Refusals: negative tree count, negative feature budget

## 4. Documentation
- [x] 4.1 `Docs/ml.md`, `skills/insyra/`, changelogs in both languages
