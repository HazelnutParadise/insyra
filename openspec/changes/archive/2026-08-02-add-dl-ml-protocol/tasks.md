# Tasks

## 1. Protocol adapters
- [x] 1.1 `BindRegressor(model, inputName, features)`: DataTable → input tensor by named columns; output tensor → DataList; structural `ml.Model` without importing `ml`
- [x] 1.2 `BindClassifier(..., classes)`: label = argmax, `Classes()`, `PredictProba` as a probability table in class order
- [x] 1.3 Refusals: unknown input name, feature/width mismatch, class-count/output-width mismatch — each naming the offending thing
- [x] 1.4 `mltest.RunConformance` green for both adapters (from dl's external test package)

## 2. The ai.onnx.ml domain
- [x] 2.1 Enumerate the exact operator set from `ml/onnx_export.go` — implement what the exporter writes, not a guess
- [x] 2.2 `LinearRegressor`, `LinearClassifier` (post_transform NONE/LOGISTIC), `TreeEnsembleRegressor`, `TreeEnsembleClassifier` (including the binary single-score convention `ml` writes), `Scaler`
- [x] 2.3 Standard-domain operators the pipeline exports need: `Concat`, `Unsqueeze`, `Gather`, `OneHotEncoder`, `LabelEncoder`, plus `Cast`/string handling as the exporter requires

## 3. Round-trip closure
- [x] 3.1 For every family `ml` exports (linear, ridge, lasso, WLS, logistic, both trees, both forests, both boosters): export → `dl` load → run → compare against the fitted model's own predictions AND against `onnxruntime`, both within f32 tolerance
- [x] 3.2 The exported-pipeline case with preprocessing
- [x] 3.3 One-op parity harness rows for each new operator where the Python `onnx` builder can express it

## 4. Documentation
- [x] 4.1 `Docs/dl.md` and `Docs/ml.md` cross-reference the closed loop; changelogs both languages; `skills/insyra/` updated
