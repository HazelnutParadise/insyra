# Tasks: fix-api-review-batch-2

## 1. Tests first

- [x] 1.1 `datatable_batch2_test.go`：T-1 三個 panic、T-2 Data 複本、T-4/T-5 Filter 別名與空表方法、FilterRows jagged、T-6 兩個情境、T-8 Transpose、T-9 ChangeRowName、T-3 AppendRowsByColIndex、T-7 int64、T-10 Mean、T-12 CSV 時間往返、E-1 CCL 失敗後鏈式呼叫
- [x] 1.2 `stats/test_input_test.go`：七個檢定與 CalculateMoment 遇 nil 回含列號錯誤
- [x] 1.3 `mkt/rfm_batch2_test.go`：非數值金額不 panic；RFM／CAI 輸出排序
- [x] 1.4 `cli/env/manager_name_test.go`：`../outside` 拒絕、正常名稱通過
- [x] 1.5 `lp`：additional info 列順序；`datafetch`：cache Set 後無 .tmp 殘留

## 2. Implementation

- [x] 2.1 `datatable.go`：GetElementByNumberIndex／SetRowToColNames／SetColToRowNames 邊界；Data/ToMap 複本；DropRowsByIndex 正規化；Transpose 列名；AppendRowsByColIndex 補欄；DropColsContainNumber／DropRowsContainNumber 用 IsNumeric；Mean 分母
- [x] 2.2 `datatable_filters.go`：clone helper、NewDataTable()、整表列數
- [x] 2.3 `datatable_rowname.go`：ChangeRowName 走 safeRowName
- [x] 2.4 `datatable_csv.go`：time.Time 用 RFC3339Nano
- [x] 2.5 `datatable_ccl.go`：具名回傳，移除 resultDtChan
- [x] 2.6 `stats/ttest.go`、`ztest.go`、`ftest.go`、`moments.go`：numericSlice
- [x] 2.7 `mkt/rfm.go`、`mkt/cai.go`：ToFloat64Safe；排序輸出
- [x] 2.8 `cli/env/manager.go`：名稱驗證
- [x] 2.9 `lp/lp.go`：固定順序；`datafetch/geocoding.go`：tmp+rename

## 3. Docs, changelog, follow-ups

- [x] 3.1 `Docs/DataTable.md`（Data 複本、Filter 獨立、DropRowsByIndex、Transpose、ChangeRowName、ToCSV 時間）、`Docs/stats.md`（七個檢定的輸入條件）、`Docs/mkt.md`（RFM 跳過規則與排序）、`Docs/cli-dsl.md`（環境名稱規則）
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：Core、`stats`、`mkt`、CLI、`lp`、`datafetch` 條目，行為改變標 breaking
- [x] 3.3 `api-review.md`：T-1、T-2、T-3、T-4、T-5、T-6、T-7、T-8、T-9、T-10、T-12（時間部分）、E-1、ST-1、ST-2、MK-1、MK-2、CL-1、LP-2（順序部分）、DF-6 標已修正

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`golangci-lint run` 無新錯誤
- [x] 4.2 `openspec validate fix-api-review-batch-2 --strict` 通過
