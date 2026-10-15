# Tasks: datatable-equality

## 1. Tests first

- [x] 1.1 新測試 `datatable_equality_test.go`：名稱不同但格子相同、型別不同的一格、NaN、形狀不同、短欄視為 nil、clone、只差一個列名、只差一個欄名、只差表名、同一張表、nil 參數（舊程式碼沒有這兩個方法，編譯失敗即為紅）

## 2. Implementation

- [x] 2.1 新檔 `datatable_equality.go`：新增 `IsEqualTo`、`IsTheSameAs`，以 `AtomicDoAll` 一起鎖兩張表
- [x] 2.2 `interfaces.go`：`IDataTable` 加入兩個方法

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`：新增比較一節
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core
- [x] 3.3 `delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate datatable-equality --strict`
