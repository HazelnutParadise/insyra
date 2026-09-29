# Tasks: ml-one-onnx-export-name

## 1. Tests first

- [x] 1.1 新測試：`ml.ExportONNX` 第二個參數型別是 `ml.Model`；`WriteONNX`、`DecisionTreeClassifierOptions`、`DecisionTreeRegressorOptions` 的 doc comment 有指名替代名稱的 `Deprecated:` 段落與移除說明（舊程式碼缺少，即為紅）
- [x] 1.2 新測試：`WriteONNX` 與 `ExportONNX` 對同一個模型寫出相同位元組；`WriteONNX` 對字串與 nil 回傳錯誤且不寫入；`ExportONNX(w, nil)` 回傳錯誤且不寫入；以 `DecisionTreeClassifierOptions` 與 `DecisionTreeOptions` fit 出相同的樹

## 2. Implementation

- [x] 2.1 `ml/onnx_export.go`：`ExportONNX(w io.Writer, fitted Model)`，nil 時回傳錯誤；`WriteONNX(w io.Writer, fitted any)` 標 Deprecated，`Model` 轉交 `ExportONNX`，其他值回傳原本的錯誤
- [x] 2.2 `ml/decision_tree.go`：兩個 options 別名標 Deprecated

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/ml.md`：ONNX 匯出段落寫明參數是 `Model`、`WriteONNX` 已 Deprecated；決策樹段落只寫 `DecisionTreeOptions`
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`ml` 與 `nn`，參數型別改變標 BREAKING
- [x] 3.3 `AGENTS.md`：下一版移除 Deprecated 名稱的 follow-up
- [x] 3.4 `api-review.md`：ML-2 標為已修正；`delivery-status.md`

## 4. Verification

- [ ] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate ml-one-onnx-export-name --strict`
