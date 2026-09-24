# Tasks: make-errors-non-terminating

## 1. Tests first

- [x] 1.1 `error_philosophy_test.go`：LogFatal 不結束程序、`SetPanicOnError` 可 recover、等級順序、`LogError`
- [x] 1.2 `instance_error_test.go`：Err() 黏性、`PopErr()` 讀後清、`Clone()` 乾淨、查無結果不設 Err()
- [x] 1.3 `chainable_nil_test.go`：八個轉換失敗回空 list 並在接收者記錄錯誤，串接不 panic
- [x] 1.4 `isr/error_test.go`：`DT.From` 讀檔失敗、`Col`／`Row`／`Push` 型別錯、`UseDL`／`UseDT` 皆回可用物件
- [x] 1.5 `error_buffer_bound_test.go`：5000 筆後不超過上限；pop 全域不影響實例
- [x] 1.6 `gplot`：`SaveChart` 回錯誤；零值 `HistogramConfig` 不 panic

## 2. Implementation

- [x] 2.1 `config.go`：`SetPanicOnError`／`GetPanicOnError`，`SetDontPanic` 轉為 Deprecated 別名；`LogLevelError`、`String()`
- [x] 2.2 `logger.go`：`LogError`；`LogFatal` 只記錄；panic 由 `SetPanicOnError` 決定
- [x] 2.3 `error_buffer.go`：有上限的 push；九個存取函式標 Deprecated
- [x] 2.4 `datalist.go`／`datatable.go`：`setError` 黏性、`PopErr()`、`fail()` helper；逐一分類既有 `warn` 呼叫點
- [x] 2.5 八個回 nil 的 `DataList` 轉換改回空 list；`interfaces.go` 加 `PopErr`
- [x] 2.6 `isr/dt.go`、`isr/dl.go`、`isr/use.go`：19 個 LogFatal 點改記錄錯誤
- [x] 2.7 `gplot/save_chart.go` 回 `error`；`gplot` 三個 `panic(err)`、`plot` 兩個 panic 與 `plot/radar.go` 的 LogFatal 改記錄
- [x] 2.8 `lp/init.go`、`lp/lp.go`、`py/pyresult.go`：LogFatal 改為記錄並讓上層回報
- [x] 2.9 `cli/commands/timeseries.go`：改檢查 `Err()` 而非 nil

## 3. Docs, changelog, review ledger

- [x] 3.1 `Docs/Configuration.md`（哲學、`SetPanicOnError`、等級）、`Docs/DataList.md`／`DataTable.md`（黏性、`PopErr`、不回 nil）、`Docs/isr.md`（改寫 Error Handling 一節）、`Docs/gplot.md`（`SaveChart` 簽名）、`README.md`／`README_TW.md`
- [x] 3.2 `skills/insyra/SKILL.md`：錯誤處理段落
- [x] 3.3 `AGENTS.md`：follow-up 名稱改為 `SetPanicOnError`；`Key Conventions` 補兩條規則
- [x] 3.4 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core、`isr`、`gplot`、`lp`、`py`，BREAKING 標示
- [x] 3.5 `api-review.md`：K-1、K-3、D-3、D-5、IN-7 標已修正；關閉 #205、#206、#216

## 4. Verification

- [x] 4.1 `go test ./...` 全綠；`go test -race` core／isr／cli；`golangci-lint run` 0 issues
- [x] 4.2 `openspec validate make-errors-non-terminating --strict` 通過
