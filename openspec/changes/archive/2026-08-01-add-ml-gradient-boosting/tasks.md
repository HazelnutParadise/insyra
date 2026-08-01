# Tasks

## 1. Implementation
- [x] 1.1 Regressor: mean base value, residual-fitted stages, learning-rate shrinkage, early stop on zero residuals
- [x] 1.2 Classifier: prior log-odds base, probability residuals, Newton-step leaf replacement, sigmoid prediction
- [x] 1.3 Options with scikit-learn's defaults; tree options routed through the single-tree defaulting — the same zero-MaxBins trap the forest hit
- [x] 1.4 Refusals: negative stages, non-positive learning rate, single-class and multiclass targets

## 2. Verification
- [x] 2.1 Determinism with no seed involved
- [x] 2.2 200 stages fit training data no worse than 5, and R² ≥ 0.95 on a near-deterministic surface
- [x] 2.3 Conformance for both; probabilities sum to one
- [x] 2.4 Accuracy agreement with scikit-learn on separable data, behind the reference gate
- [x] 2.5 The multiclass refusal names the limit

## 3. Documentation
- [x] 3.1 `Docs/ml.md`, `skills/insyra/`, changelogs in both languages
