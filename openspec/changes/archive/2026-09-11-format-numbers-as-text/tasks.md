# Tasks: format-numbers-as-text

## 1. Tests first

- [x] 1.1 `internal/utils/text_test.go`：`FloatText`／`ValueText` 的邊界值（999999、1e6、1.5e6、1e20、1e21、1e-6、9.99e-7、1e-7、1e-5、0、-0、NaN、±Inf、float32）
- [x] 1.2 根套件：CCL 營收字串、`LEN(1000000)`、`TOSTR`；`ToCSV` 寫 `1500000` 且讀回逐值相等；`ToStringSlice`；one-hot 欄名；pivot 欄名；nil 標籤不變；排序不變

## 2. Implementation

- [x] 2.1 `internal/utils`：`FloatText`、`ValueText`
- [x] 2.2 `internal/ccl/stdlib_string.go` `toString`
- [x] 2.3 `datatable_csv.go`、`datalist.go` `ToStringSlice`
- [x] 2.4 `datatable_encode.go` `oneHotCategoryColumnName`、`datatable_pivot.go` `pivotColLabel`

## 3. Docs, changelog

- [x] 3.1 `Docs/CCL.md`、`Docs/DataTable.md`、`Docs/DataList.md`、`skills/insyra/references/ccl-operators.md`
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：標 BREAKING
- [x] 3.3 `AGENTS.md`：刪除已解決的 follow-up；`api-review.md` 更新 CCL-35 註記

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate format-numbers-as-text --strict` 通過
