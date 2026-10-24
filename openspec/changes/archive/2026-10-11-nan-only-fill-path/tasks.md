# Tasks: nan-only-fill-path

## 1. Tests first

- [x] 1.1 CLI：規格的四個情境（`missing nan|nil` 與 `limit` 的搭配，包含表格欄位）
- [x] 1.2 根套件：`ReplaceNaNsWith(Clone().ClearNilsAndNaNs().Mean())` 與 `FillNaNWithMean` 逐格相同

## 2. Implementation

- [x] 2.1 `cli/commands/fillna.go`：`ffill`／`bfill` 在 `missing nan|nil` 時，跳過另一種缺值後填補再放回原位；表格欄位依 `DataTable.FillForward` 方式解析
- [x] 2.2 `datalist.go`：`FillNaNWithMean` 的 Deprecated 說明改寫成保留行為的替代寫法（保留於 0.3.x，在 0.4 移除）

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`：`FillNaNWithMean` 的替代寫法
- [x] 3.2 `Docs/cli-dsl.md`：`missing` 與 `limit` 的關係
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：更新條目

## 4. Verification

- [x] 4.1 驗證通過：`go test ./...` 通過
