# Tasks: fix-api-review-batch-3

## 1. Tests first

- [x] 1.1 `batch3_core_test.go`：Config setter 無 race、error hook 依序、DetectEncoding 邊界、IsEqualTo NaN、ClearNilsAndNaNs 單趟、Update 記自己的名字、FindColsIfContains 無警告、AppendRowsByColName 欄序固定
- [x] 1.2 `plot`：SavePNG 預設不上傳（錯誤文字提示 opt-in）
- [x] 1.3 `csvxl/errors_batch3_test.go`：錯誤可 `errors.Is`；輸出目錄 0755
- [x] 1.4 `parquet/write_atomic_test.go`：失敗不留截斷檔
- [x] 1.5 `stats/type_guard_test.go`：nil 介面不 panic
- [x] 1.6 `cli/commands/registry_race_test.go`：並行 Register／Dispatch

## 2. Implementation

- [x] 2.1 `plot/save_chart.go`：線上退回改 opt-in，doc 說明上傳內容
- [x] 2.2 `config.go`、`logger.go`、`utils.go`：atomic 欄位與 getter；`error_buffer.go`：單一 worker + 有上限佇列；`init.go`／`cli/repl/repl.go`：橫幅移到 REPL
- [x] 2.3 `utils.go`：DetectEncoding 先裁掉不完整 rune 再驗 UTF-8
- [x] 2.4 `datalist.go`：filterCells 單趟清理、equalCell NaN 相等、findFirstIndex、Update 名稱、doc 修正
- [x] 2.5 `datatable.go`：FindColsIfContains* 無警告、containsSubstring=strings.Contains、Count／Clone 平鋪、AppendRowsByColName 欄名排序
- [x] 2.6 `csvxl/convert.go`、`convertDir.go`：`%w`、0o755
- [x] 2.7 `parquet/*.go`：Write tmp+rename；LogWarning；ApplyCCL doc
- [x] 2.8 `stats/asdatalist.go` 與 13 個斷言點
- [x] 2.9 `cli/commands/registry.go`：RWMutex；`mkt/rfm.go`、`mkt/cai.go`：LogDebug
- [x] 2.10 doc comments：`atomic.go`、`accel/types.go`、`accel/executor.go`、`parallel/parallel_computing.go`、`datatable_from_sql.go`、`datatable_to_sql.go`

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/plot.md`（SavePNG 預設）、`Docs/DataList.md`（IsEqualTo NaN）、`Docs/DataTable.md`（AppendRowsByColName 欄序）、`Docs/csvxl.md`、`Docs/parquet.md`、`skills/use-insyra-cli/references/cli-command-usage.md`（PNG 需本機 Chrome）
- [x] 3.2 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：Core、`plot`（BREAKING）、`stats`、`csvxl`、`parquet`、CLI、`mkt`
- [x] 3.3 `api-review.md`：PL-1、K-5、K-6、K-10、K-16、K-18、D-9、D-11、D-15（部分）、D-17、D-19、T-18、T-19、T-20（doc）、C-5、C-7、Q-3、Q-4、Q-5、P-3、ST-3、CL-2（Registry）、MK-4、E-4、E-5、AC-2 標已修正

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`go test -race` core／cli／stats 無 race；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate fix-api-review-batch-3 --strict` 通過
