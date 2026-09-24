# Tasks: fix-api-review-batch-1

## 1. Tests first

- [x] 1.1 `datalist_test.go`（或新檔 `datalist_numeric_test.go`）：ReplaceLast 三個情境；Normalize/Standardize/ClearOutliers/Difference/FillNaNWithMean 遇字串不改資料且設 Err、遇 nil/NaN 保留；Rank 拒字串、NaN 名次為 NaN；ExponentialSmoothing/DoubleExponentialSmoothing 拒字串；`datalist_interpolation_test.go` 六個插值拒 nil 並指出位置
- [x] 1.2 `stats/skewness_test.go`、`stats/kurtosis_test.go`：nil 與字串格子回含 `sample` 與 `row 2` 的錯誤
- [x] 1.3 `read_test.go`：`ReadJSON_File` 大整數保留 int64、單一物件成一列、與 `ReadJSON(bytes)` 逐格相同
- [x] 1.4 `csvxl/convert_test.go`：同名工作表 3 列→1 列後只剩 1 列；只有一張工作表時可替換；`ExcelToCsv` 錯誤路徑仍 Close
- [x] 1.5 `parquet/api_test.go`（新檔）：寫 1000 列檔案，`MaxValues: 10` 回錯、`1000` 成功、兩個 row group 選一個時以所選為準

## 2. Implementation

- [x] 2.1 新增 `datalist_numeric.go`：`numericCells(data []any, allowMissing bool) ([]float64, int, bool)`
- [x] 2.2 `datalist.go`：修 `ReplaceLast`；`Normalize`、`Standardize`、`ClearOutliers`、`Difference`、`FillNaNWithMean` 改先掃後寫，移除 `conv.ParseF64` 與 recover；`Rank`、`ExponentialSmoothing`、`DoubleExponentialSmoothing` 改走 `numericCells`；檢查 `datalist_notatomic.go` 的 `replaceLast_notAtomic`
- [x] 2.3 `datalist_interpolation.go`：六個方法改走 `numericCells(data, false)`，失敗設 Err 回 NaN
- [x] 2.4 `stats/skewness.go`、`stats/kurtosis.go`：`SliceToF64` 改 `numericValues(d, "sample")`
- [x] 2.5 `read.go`：`ReadJSON_File` 改為 `os.ReadFile` + `ReadJSON`
- [x] 2.6 `csvxl/convert.go`、`csvxl/convertDir.go`：同名工作表先刪再建（含單一工作表的暫存表路徑）；三處 `defer f.Close()`
- [x] 2.7 `parquet/api.go`：`ReadColumn` 由 metadata 加總列數檢查 `MaxValues`

## 3. Docs, changelog, follow-ups

- [x] 3.1 `Docs/DataList.md`：ReplaceLast、Normalize、Standardize、ClearOutliers、Difference、FillNaNWithMean、Rank、ExponentialSmoothing、DoubleExponentialSmoothing、六個 Interpolation 各補一句非數值格子的行為
- [x] 3.2 `Docs/stats.md`（Skewness、Kurtosis 錯誤條件）、`Docs/DataTable.md`（ReadJSON_File 數字型別）、`Docs/csvxl.md`（AppendCsvToExcel 替換語意）、`Docs/parquet.md`（MaxValues 由 metadata 檢查）
- [x] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased`：`### Core` 段末、`### `stats``、`### `csvxl``、`### `parquet`` 各加條目，行為改變依過去 release note 標示 breaking
- [x] 3.4 `AGENTS.md` follow-up「`ToF64Slice` still fabricates zeros」：Where 移除 `datalist_interpolation.go`，What 補一句本變更把 Rank/smoothing/interpolation 移離該路徑
- [x] 3.5 `api-review.md`：D-1、D-2、D-4（High 部分）、K-2、K-9、C-3、C-4、Q-1 標「已修正（fix-api-review-batch-1）」

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 無新錯誤
- [x] 4.2 `openspec validate fix-api-review-batch-1 --strict` 通過
