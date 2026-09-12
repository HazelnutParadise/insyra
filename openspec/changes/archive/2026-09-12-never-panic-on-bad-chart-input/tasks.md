# Tasks: never-panic-on-bad-chart-input

## 1. 先重現

- [x] 1.1 為八個 panic 各寫一個測試，確認修正前是紅的。

## 2. gplot

- [x] 2.1 `bar.go`：`XAxis` 為空時不呼叫 `NominalX`。
- [x] 2.2 `function.go`：函式為 nil 時記錄並回傳 nil。
- [x] 2.3 `heatmap.go`：列長度不一致時記錄是哪一列並回傳 nil。
- [x] 2.4 同檔案：`Colors` 非正值時套用預設值。

## 3. plot

- [x] 3.1 `bar.go`、`line.go`、`wordcloud.go`、`boxplot.go`：略過 nil 的資料清單元素；全部都是 nil 時回傳 nil。
- [x] 3.2 `save_chart.go`：`SavePNG` 先檢查副檔名。

## 4. lpgen 與 utils

- [x] 4.1 `lingo.go`：括號順序不合理的宣告被略過。
- [x] 4.2 `internal/utils/utils.go`：`TruncateString` 的負寬度視為 0。

## 5. 收尾

- [x] 5.1 1.1 的測試全部轉綠；`go test ./...`、`go test -race`（動到的套件）、`golangci-lint run`。
- [x] 5.2 兩份 CHANGELOG 的 `### gplot`、`### plot`、`### lpgen` 條目。
- [x] 5.3 `Docs/gplot.md` 說明 `XAxis` 省略時的行為。
- [ ] 5.4 `delivery-status.md` 里程碑。
