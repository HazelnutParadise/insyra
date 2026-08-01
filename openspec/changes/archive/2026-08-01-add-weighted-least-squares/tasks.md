# Tasks

## 1. `stats`
- [x] 1.1 `WeightedLinearRegression` — weighted normal equations, inference, weighted R², residuals, `Predict`
- [x] 1.2 Weights validated strictly positive and finite, refusals naming the row
- [x] 1.3 Uniform weights pinned equal to `LinearRegression`

## 2. Verification
- [x] 2.1 `wls` method in the Python baseline; coefficients, SE, t, p, R² and predictions against statsmodels
- [x] 2.2 The Python gate extended to require statsmodels alongside scikit-learn

## 3. `ml`
- [x] 3.1 `FitWeightedLinearRegression(x, y, weights)`, conformance-checked
- [x] 3.2 The CrossValidate limitation documented where the function is documented

## 4. Documentation
- [x] 4.1 `Docs/stats.md`, `Docs/ml.md`, `skills/insyra/`, changelogs in both languages
- [x] 4.2 Record the two open design questions (tree weights vs the fixed-point contract; a weights channel in the protocol) in the proposal and the handoff surface
