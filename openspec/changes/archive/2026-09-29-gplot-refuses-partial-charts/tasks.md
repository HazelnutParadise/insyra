# Tasks: gplot-refuses-partial-charts

## 1. Tests first

- [x] 1.1 `gplot/chart_errors_test.go`：line 一條長度對、一條不對時回傳 nil 圖表與點名較短那條的錯誤；step 同時有空清單與含 NaN 的清單時，錯誤點名兩條並各附原因；scatter 一個正常系列加一個空系列時回傳錯誤；以上都不寫任何 log。nil 清單混在真實清單中時仍回傳圖表並警告
- [x] 1.2 在目前程式上執行 1.1，記錄失敗
  - 紀錄：`TestAnySeriesThatCannotBeDrawnFailsTheChart` 的 4 個子測試全部失敗，都是 `a chart missing a series came back with no error`；`TestANilListAmongRealOnesIsStillDropped` 通過（行為不變）

## 2. Implementation

- [x] 2.1 `gplot/line.go`：任何系列畫不出來就回傳錯誤，錯誤列出每一條與原因，不再記警告；`finishSeries` 改為只產生錯誤
- [x] 2.2 `gplot/scatter.go`、`gplot/step.go`：同一規則；三個建構函式的 doc comment 更新
- [x] 2.3 更新既有測試：`TestOneDrawableSeriesIsEnough`、`TestCreateLineChart_SkipsAMismatchedSeries`、scatter 的「an empty series beside a real one」

## 3. Docs, changelog

- [x] 3.1 `Docs/gplot.md`：Line、Step、Scatter 各節與「A series that cannot be drawn is skipped」一節改寫成新規則
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：改寫 `gplot` 未發布的「(behaviour)」條目，直接描述相對 v0.3.3 的最終行為
- [x] 3.3 `delivery-status.md`：Latest Milestones 最上方一則

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate gplot-refuses-partial-charts --strict`
