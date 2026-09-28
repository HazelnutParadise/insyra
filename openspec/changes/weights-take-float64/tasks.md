# Tasks: weights-take-float64

## 1. Tests first

- [x] 1.1 新測試檔：`[]float64` 權重的 `WeightedMean`、`WeightedMovingAverage` 結果；長度不符的失敗形狀；非數值元素連同權重略過；以 `reflect` 檢查兩個方法的權重參數型別是 `[]float64`（先紅：型別檢查失敗）
- [x] 1.2 `datalist_test.go` 的 `TestDataListWeightedMean` 改傳 `[]float64`；`utils_test.go` 移除 `core-utils-cleanup` 加的 `TestWeightedMethodsReportUnreadableWeights`（傳 `42`、`"ab"`、`DataList` 當權重，新簽名下無法編譯），`exported-functions` 規格的對應情境只留 `stats.Skewness`

## 2. Implementation

- [x] 2.1 `datalist.go`：兩個方法改收 `[]float64`，移除 `ProcessData` 與權重的 `ToFloat64Safe` 判斷
- [x] 2.2 `interfaces.go`：`IDataList` 的兩個簽名

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`：`WeightedMean`、`WeightedMovingAverage` 的簽名、參數說明與範例
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：Core 新增 BREAKING；`ProcessData` 條目中「權重讀不了時回報成權重問題」那句移除，因為權重已不經過 `ProcessData`
- [x] 3.3 `api-review.md`：D-10 標為已修正
- [x] 3.4 `delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`、`openspec validate weights-take-float64 --strict`
