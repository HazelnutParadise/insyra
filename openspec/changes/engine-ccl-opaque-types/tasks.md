# Tasks: engine-ccl-opaque-types

## 1. 測試先行

- [x] 1.1 `engine/ccl/opaque_types_test.go`：`Context` 的 15 個方法、`CCLNode` 是沒有公開欄位的 struct 且零值被拒絕、`MapContext` 沒有公開欄位且照常運作；在舊程式上編譯失敗
- [x] 1.2 `engine/ccl/ccl_test.go`：三處 `nil` 比較改成和 `ccl.CCLNode{}` 比較

## 2. 實作

- [x] 2.1 `engine/ccl/ccl.go`：`CCLNode` 改成包住內部節點的 struct，所有節點函式包裝與拆開，零值回錯；`CompiledStatement` 改成獨立 struct
- [x] 2.2 `engine/ccl/map_context.go`：包裝內部 `MapContext`，以委派實作 `Context`，零值回錯不 panic
- [x] 2.3 `Context` 說明寫明方法集固定；`internal/ccl/context.go` 加維護者說明

## 3. 文件與紀錄

- [x] 3.1 `engine/README.md` CCL 章節
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### Core` 段落末尾新增條目，兩條標 BREAKING
- [x] 3.3 `api-review.md`：EN-1、CCL-29 標為已修正
- [x] 3.4 `delivery-status.md`：Latest Milestones 最上方新增條目
- [x] 3.5 `Docs/CCL.md` 與 skills：不需要修改，它們沒有提到這些型別

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate engine-ccl-opaque-types --strict`
