# Tasks

## 1. Linear families
- [x] 1.1 Ridge, lasso and WLS through the LinearRegressor path, in both the direct dispatch and the pipeline-estimator dispatch
- [x] 1.2 Exporter methods on the three model types

## 2. Ensembles
- [x] 2.1 Multi-tree and leaf-scale support in the tree-ensemble builder; base values attribute
- [x] 2.2 Forest classifier and regressor: per-tree ids, 1/T scaling
- [x] 2.3 Boosted regressor: learning-rate scaling, mean base value
- [x] 2.4 Boosted binary classifier: log-odds scores with the runtime's logistic transform — the first form wrote both classes' weights and the runtime's binary path returned label 1 for every row while the probabilities were exactly right; both binary classifiers now use the single-score convention (one entry per leaf on class slot 0, complement and half-threshold label computed by the runtime), verified by execution
- [x] 2.5 Exporter methods on the four model types

## 3. Verification
- [x] 3.1 Round-trip cases for all seven families against the real runtime
- [x] 3.2 Existing exports unchanged: the full suite still passes

## 4. Documentation
- [x] 4.1 `Docs/ml.md` ONNX section's supported list; skill; changelogs in both languages
