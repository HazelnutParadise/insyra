# Tasks: lp-never-returns-a-nil-table

## 1. 先重現

- [x] 1.1 測試：`SolveFromFile` 的每一條失敗路徑，斷言兩個回傳值都不是 nil 且 `Show()` 不 panic。
- [x] 1.2 測試：`SolveModel` 的每一條不需要 GLPK 的失敗路徑同上。
- [x] 1.3 測試：`parseGLPKOutputFromFile` 讀不到檔案時回傳可用的空表而不是 nil。

## 2. 修正

- [x] 2.1 `lp/lp.go`：加一個回傳「空表 + 記錄原因」的小 helper。
- [x] 2.2 `SolveFromFile` 的五條失敗路徑改用它；`(nil, nil)` 的兩處也補上資訊表。
- [x] 2.3 `SolveModel` 的六條失敗路徑同上。
- [x] 2.4 `parseGLPKOutputFromFile` 改回傳空表。
- [x] 2.5 求解成功但結果檔讀不到時，資訊表的 `Status` 改成 `Error`。

## 3. 收尾

- [x] 3.1 測試全綠；`go test ./...`、`golangci-lint run`。
- [x] 3.2 `Docs/lp.md` 的範例先檢查 `Err()`，並說明回傳值不會是 nil。
- [x] 3.3 兩份 CHANGELOG 的 `lp` 條目，標明 `result == nil` 的檢查不再成立。
- [x] 3.4 `delivery-status.md` 里程碑；`api-review.md` 的 LP-2 註記這次處理了哪一部分。
