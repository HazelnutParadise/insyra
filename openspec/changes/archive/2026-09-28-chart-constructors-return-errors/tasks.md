# Tasks: chart-constructors-return-errors

## 1. Tests first

- [x] 1.1 `plot/chart_errors_test.go`：13 個建構函式的回傳型別第二個是 `error`（以 `reflect` 檢查）；每一種文件寫明會失敗的輸入回傳 nil 圖表與開頭是 `plot: CreateXxx:` 的錯誤；nil 清單混在真實清單中時回傳圖表與 nil 錯誤；`CreateBoxPlot` 只有空系列時錯誤說沒有系列有資料；`CreateGaugeChart` 錯誤為 nil；bar／line 遇到一格文字時 Y 軸變成類別、數字字串畫成 0；box plot 略過非數值格
- [x] 1.2 `gplot/chart_errors_test.go`：7 個建構函式回傳 `(*plot.Plot, error)`，資料參數型別為 `insyra.IDataList`／`...insyra.IDataList`／`insyra.IDataTable`／`...ScatterSeries`（以 `reflect` 檢查）；空清單、NaN、nil、X/Y 長度不同、所有系列都畫不出來各自回傳錯誤；一個系列畫得出來時回傳圖表；bar 對 `1, "2", nil, "abc"` 畫成 1, 0, 0, 0；heat map 對 `int8`、`uint16`、`float32` 欄畫出原值
- [x] 1.3 在舊程式上執行 1.1、1.2，記錄失敗（編譯失敗即為紅）
  - 紀錄：舊程式上 `go test ./plot ./gplot` 兩個套件都 build failed，`gplot` 報 `undefined: ScatterSeries` 與 `not enough return values: have (*plot.Plot), want (*plot.Plot, error)`，`plot` 報 `assignment mismatch: 2 variables but CreateBarChart returns 1 value`。heat map 讀 `int8` 的測試另以對照組確認：舊的轉換把 `int8`、`uint16` 欄畫成 0，SVG 與新結果不同

## 2. Implementation

- [x] 2.1 `plot/*.go`：13 個建構函式回傳 `(chart, error)`，失敗時回傳錯誤、不再記警告；`CreateBoxPlot` 逐一警告被丟棄的系列，錯誤說明沒有系列有資料；每個建構函式的 doc comment 寫明失敗條件與非數值的畫法
- [x] 2.2 `gplot/*.go`：新簽名與 `ScatterSeries`；移除 `*DataList`／`IDataList` 重複的輔助函式；line、step、scatter 畫不出任何系列時回傳錯誤；heat map 以 `ToF64Slice` 讀每一欄；doc comment 寫明非數值畫成 0
- [x] 2.3 `cli/commands/plot.go`：回報建構函式的錯誤
- [x] 2.5 對抗式審查的修正：`CreateHistogram` 拒絕 NaN／±Inf，`CreateHeatmapChart` 拒絕 ±Inf 與全為 NaN 的表格，`CreateFunctionPlot` 拒絕非有限的邊界；失敗時不記任何 log（跳過的 nil 清單、系列只在回傳圖表時才警告，空清單不經 `ToF64Slice`）；line、step、scatter 的錯誤寫出每個系列與原因；測試改為斷言失敗時 log 為空，並在 amd64（Rosetta）與 arm64 各跑一次
- [x] 2.4 更新既有測試：`plot/charts_test.go`、`plot/no_panic_test.go`、`plot/heatmap_point_test.go`、`gplot/charts_test.go`、`gplot/no_panic_test.go`、`gplot/step_test.go`、`gplot/save_chart_test.go`；新增 `cli/commands/plot_test.go`，驗證 CLI 回報建構函式的錯誤

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/plot.md`：每個建構函式的簽名與回傳值、失敗條件、非數值的畫法；「A chart that cannot be built is nil」一節改寫
- [x] 3.2 `Docs/gplot.md`：新簽名、`ScatterSeries`、每個範例、失敗條件、非數值畫成 0
- [x] 3.3 `Docs/cli-dsl.md`：`plot` 對非數值格的畫法
- [x] 3.4 `Docs/tutorials/` 三篇圖表教學的呼叫
- [x] 3.5 `skills/insyra/SKILL.md`：錯誤形狀的原則涵蓋圖表建構函式；非數值格一句與實測一致
- [x] 3.6 `CHANGELOG.md`／`CHANGELOG_TW.md`：`plot`、`gplot`、CLI 各一則，標 BREAKING
- [x] 3.7 `api-review.md`：PL-3 標為已修正
- [x] 3.8 `delivery-status.md`：Latest Milestones 最上方一則
- [x] 3.9 `AGENTS.md`：更新提到 `plot.Create...` 回傳 nil 的 follow-up；`CreateBoxPlot` 訊息一項移除；新增 NaN／±Inf 讓 `plot` 圖表空白的 follow-up

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate chart-constructors-return-errors --strict`
