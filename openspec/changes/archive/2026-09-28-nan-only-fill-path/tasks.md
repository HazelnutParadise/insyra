# Tasks: nan-only-fill-path

## 1. Tests first

- [x] 1.1 CLI：規格的四個情境（先紅：前兩個與表格情境在舊程式失敗，第三個新舊都過）；無 `limit` 的既有測試不改
- [x] 1.2 根套件：`ReplaceNaNsWith(Clone().ClearNilsAndNaNs().Mean())` 與 `FillNaNWithMean` 逐格相同

## 2. Implementation

- [x] 2.1 `cli/commands/fillna.go`：`ffill`／`bfill` 在 `missing nan|nil` 時，拿掉另一種缺值後填補再放回原位；其餘策略維持「補完再還原」
- [x] 2.2 `datalist.go`：`FillNaNWithMean` 的 Deprecated 說明改寫成保留行為的替代寫法

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`：`FillNaNWithMean` 的替代寫法（mean、median）
- [x] 3.2 `Docs/cli-dsl.md`：`missing` 與 `limit` 的關係
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：CLI
- [x] 3.4 `delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate nan-only-fill-path --strict`
