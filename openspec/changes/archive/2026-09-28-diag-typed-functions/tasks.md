# Tasks: diag-typed-functions

## 1. 四個具型別的函式

- [x] 1.1 先寫會失敗的測試：`DiagOf`（方陣、長方矩陣、nil 介面、nil `*mat.Dense`）、`DiagMatrix`、`DiagMatrixSize`（補 0、長方單位矩陣、`v` 過長、大小小於 1）、`IdentityMatrix`（含 0 與負數）、不修改輸入；在舊程式上編譯失敗於函式未定義
- [x] 1.2 `stats/diag.go` 實作四個函式；1.1 通過

## 2. Diag 標為 Deprecated

- [x] 2.1 先寫會失敗的測試：`Diag(0)`、`Diag([]float64{})`、`Diag(-1)` 回傳錯誤而不 panic；`Diag` 的各種既有用法與對應的新函式結果相同；在舊程式上 `Diag(0)` panic
- [x] 2.2 `Diag` 加上 Deprecated 註解與各用法的對應，在建立矩陣前檢查大小；2.1 通過

## 3. 驗證

- [x] 3.1 `gofmt`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）通過

## 4. 文件

- [x] 4.1 `Docs/stats.md` 的 Matrix Operations 改寫為四個新函式，`Diag` 標為 Deprecated 並附對應表；範例實際編譯執行過
- [x] 4.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md` 的 `## Unreleased` 在 `` ### `stats` `` 末尾新增條目（新函式、`Diag` Deprecated、不再 panic）
- [x] 4.3 `api-review.md` 的 ST-9：`Diag` 部分標為已修正（diag-typed-functions），問題清單同步
- [x] 4.4 `delivery-status.md` 的 Latest Milestones 最上方新增條目
- [x] 4.5 `AGENTS.md` 的 Follow-ups 新增「下一版移除 `Diag`」
- [x] 4.6 檢查 `skills/`：沒有提到 `Diag`，不需修改
- [x] 4.7 `openspec validate diag-typed-functions --strict` 通過
