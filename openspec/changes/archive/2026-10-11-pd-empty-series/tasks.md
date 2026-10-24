# Tasks: pd-empty-series

## 1. 測試

- [x] 1.1 `pd/pd_test.go`：`TestFromDataListEmptyIsAnEmptySeries`，空 list 轉成長度 0、`DType` 為 `interface {}` 的 Series 且沒有錯誤；空 Series 經 `ToDataList` 轉回長度 0 的 DataList

## 2. 實作

- [x] 2.1 `pd/series.go`：`FromDataList` 只讀一次 `Data()`，空 list 走 any 型別的 Series

## 3. 文件與紀錄

- [x] 3.1 `Docs/pd.md`：空 list 的新行為
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md` 的 `pd` 段落新增條目
