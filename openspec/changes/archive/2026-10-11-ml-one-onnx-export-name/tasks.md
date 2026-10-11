# Tasks: ml-one-onnx-export-name

## 1. Tests first

- [x] 1.1 新測試：`WriteONNX`、`DecisionTreeClassifierOptions`、`DecisionTreeRegressorOptions` 的 doc comment 有指名替代名稱的 `Deprecated:` 段落與移除說明（舊程式碼缺少，即為紅）
- [x] 1.2 新測試：`WriteONNX` 與 `ExportONNX` 對同一個模型寫出相同位元組；`WriteONNX` 對字串與 nil 回傳錯誤且不寫入；以 `DecisionTreeClassifierOptions` 與 `DecisionTreeOptions` fit 出相同的樹

## 2. Implementation

- [x] 2.1 `ml/onnx_export.go`：`WriteONNX(w io.Writer, fitted any)` 標 Deprecated，轉交 `ExportONNX`；`ExportONNX` 保留 `any` 參數
- [x] 2.2 `ml/decision_tree.go`：兩個 options 別名標 Deprecated

## 3. Docs and changelog

- [x] 3.1 `Docs/ml.md`：`WriteONNX` 與兩個 options 別名已 Deprecated，在 0.4 移除
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`ml`
