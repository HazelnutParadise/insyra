# Tasks: datatable-slicing

## 1. Tests first

- [x] 1.1 `datatable_slice_test.go` 與 `datatable_slice_review_test.go`：`SliceRows` 與 `SliceCols` 回傳指定範圍（含表名、欄名、列名），支援 int（負數倒數）、字母與 nil 開放端點；參差欄位補 nil；越界與反向端點回報錯誤回傳空表；結果獨立擁有資料。
- [x] 1.2 十個棄用 index 篩選方法結果與對應 slice 相同；`Headers` 與 `SetHeaders` 與 `ColNames`／`SetColNames` 相同。
- [x] 1.3 `FilterColsByColIndexLessThan("Z")` 與 `…LessThanOrEqualTo("Z")` 不崩潰並保留所有欄位；處理最大字母時不溢位。

## 2. Implementation

- [x] 2.1 `datatable_filters.go`：`SliceRows` 與 `SliceCols`；`sliceColBound`；兩個 less-than 方法修正防溢位與越界；十個 index 方法標為 Deprecated 並註明在 0.4 移除。
- [x] 2.2 `datatable_colname.go`：`Headers` 與 `SetHeaders` 標為 Deprecated 並註明在 0.4 移除。`interfaces.go`：`IDataTable` 列出 `SliceRows` 與 `SliceCols`。

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`：Slicing 章節與十二個方法的 Deprecated 說明。
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：Core。
- [x] 3.3 `api-review.md`、`delivery-status.md`。

## 4. Verification

- [x] 4.1 全套驗證。
