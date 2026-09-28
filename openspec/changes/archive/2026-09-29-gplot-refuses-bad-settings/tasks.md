# Tasks: gplot-refuses-bad-settings

## 1. Tests first

- [x] 1.1 `gplot/chart_errors_test.go`：`ErrorBars` 長度不對、含 NaN，以及 `StepStyle` 為 `"pr"` 時，都回傳 nil 圖表、以函式名稱開頭並點出設定值的錯誤，且不寫任何 log；`gplot/step_test.go` 的 `TestCreateStepChartWithInvalidStepStyle` 改為斷言錯誤；移除 `charts_test.go` 裡釘住舊行為的子測試與 `"nonsense"` 樣式
- [x] 1.2 在目前程式上執行 1.1，記錄失敗
  - 紀錄：錯誤表的三個新子測試都失敗於 `no error`，`TestCreateStepChartWithInvalidStepStyle` 也失敗

## 2. Implementation

- [x] 2.1 `gplot/bar.go`：在建圖前檢查 `ErrorBars` 的長度並建立誤差線，失敗時回傳錯誤
- [x] 2.2 `gplot/step.go`：不認得的 `StepStyle` 回傳錯誤；兩個建構函式的 doc comment 更新

## 3. Docs, changelog

- [x] 3.1 `Docs/gplot.md`：Bar 與 Step 各節、「A series that cannot be drawn fails the chart」一節最後關於 `ErrorBars` 的句子
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：`gplot` 新增一條 BREAKING (behaviour)
- [x] 3.3 `delivery-status.md`：Latest Milestones 最上方一則

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate gplot-refuses-bad-settings --strict`
