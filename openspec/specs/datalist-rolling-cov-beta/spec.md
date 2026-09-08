# datalist-rolling-cov-beta Specification

## Purpose
`RollingDataList` 的配對 reducer：滾動樣本共變異數 `Cov` 與滾動 beta `Beta`（`Cov(asset, benchmark) / Var(benchmark)`），對齊、nil 略過、`MinObs` 與截斷規則與 `Corr` 共用同一條配對迴圈。

## Requirements
### Requirement: Rolling covariance and rolling beta

`RollingDataList` SHALL 提供 `Cov(other *DataList) *DataList` 與 `Beta(other *DataList) *DataList`。`Cov` SHALL 為視窗內配對的樣本共變異數（分母 n−1）；`Beta` SHALL 為 `Cov(src, other) / Var(other)`，receiver 為資產、`other` 為 benchmark，`Var` 同為樣本變異數。兩者的索引對齊、nil 或非數值配對略過、`MinObs`、截斷到較短序列、`other == nil` 的處理 SHALL 與 `Corr` 相同。有效配對少於 2 的視窗輸出 nil；`Beta` 在 benchmark 視窗變異數為 0 時輸出 nil。

#### Scenario: Cov agrees with stats.Covariance on a full window

- **WHEN** `Window == len(src)`，兩序列皆為有限數值
- **THEN** 最後一個輸出等於 `stats.Covariance(src, other)`（1e-12 內）

#### Scenario: Beta agrees with quant.Beta on a full window

- **WHEN** `Window == len(src)`，benchmark 變異數非零
- **THEN** 最後一個輸出等於 `quant.Beta(src, other)`（1e-12 內）

#### Scenario: Rolling beta of a scaled benchmark

- **WHEN** `src[i] == 2·other[i] + c`，`Window: 5`
- **THEN** 自第 5 個位置起每個輸出為 2，前 4 個為 nil

#### Scenario: Flat benchmark window emits nil

- **WHEN** 某視窗內 `other` 的值全部相同
- **THEN** 該位置 `Beta` 輸出 nil，`Cov` 輸出 0

#### Scenario: Pairs with nil are skipped

- **WHEN** 視窗內某一列的 `src` 或 `other` 為 nil
- **THEN** 該配對不計入，有效配對數以 `MinObs` 檢查

