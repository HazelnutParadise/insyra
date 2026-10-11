# Tasks: ml-fit-returns-concrete-types

## 1. Tests first

- [x] 1.1 新測試：十個迴歸模型的 nil 指標呼叫 `Predict` 回傳錯誤；`ml.ExportONNX(w, (*ml.LinearModel)(nil))` 回傳錯誤且不寫入（舊程式碼 panic，即為紅）
- [x] 1.2 新測試：每個模型值（含 `ml.LinearModel{}`）滿足宣告 `Features` 的介面（`value_features_test.go`）
- [x] 1.3 新測試：樹與整體模型在 nil 指標上不 panic（`tree_models_nil_test.go`；這些行為在改動前就已存在，測試只是釘住）

## 2. Implementation

- [x] 2.1 `ml/models.go`：十個迴歸模型的 `Predict` 在 nil 時回傳錯誤；`Fit*` 回傳型別不變
- [x] 2.2 `ml/onnx_export.go`：`buildONNXModel` 在分派前以 `isNilPointer` 拒絕 nil

## 3. Docs and changelog

- [x] 3.1 `Docs/ml.md`：`ExportONNX` 對 nil 模型回錯誤
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`ml` 與 `nn`
