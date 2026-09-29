# Tasks: pd-empty-series

## 1. 測試先行

- [x] 1.1 `pd/pd_test.go`：空 list 轉成長度 0、`DType` 為 `interface {}` 的 Series 且沒有錯誤；空 Series 經 `ToDataList` 轉回長度 0 的 DataList；nil list 仍回傳錯誤。在舊程式上確認空 list 的測試失敗於 `empty DataList`（nil list 的測試在舊程式上本來就通過）

## 2. 實作

- [x] 2.1 `pd/series.go`：`FromDataList` 只讀一次 `Data()`，空 list 走 any 型別的 Series；1.1 通過
- [x] 2.2 `pd/series.go`、`pd/dataframe.go`：`Series`、`DataFrame`、`FromDataList`、`FromGPandasSeries`、`FromGPandasDataFrame` 的 doc comment 寫明內嵌或接收的 gpandas 型別，以及方法來自 gpandas

## 3. 文件與紀錄

- [x] 3.1 `Docs/pd.md`：說明 gpandas 依賴與內嵌型別；空 list 的新行為
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md` 的 `## Unreleased` 新增 `### pd` 段落與條目
- [x] 3.3 `api-review.md`：PD-1 標為已修正
- [x] 3.4 `delivery-status.md`：Latest Milestones 最上方新增條目
- [x] 3.5 檢查 `skills/insyra/SKILL.md`：只指向 `pd.md`，本變更不需修改

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate pd-empty-series --strict`
