# clustering-initial-centers Specification

## Purpose
`KMeans` 的初始中心選擇契約：初始中心必須是相異的資料列，與 R 一致，且沒有碰撞時不改變既有 seed 的結果。

## Requirements
### Requirement: Initial centres are distinct rows

`KMeans` 的初始中心 SHALL 為相異的資料列。抽到重複列時 SHALL 從相異列重抽，SHALL NOT 回報 "empty cluster"。抽到的中心本來就相異時 SHALL NOT 改變 RNG 消耗順序，既有 seed 的結果 SHALL 逐位不變。

#### Scenario: Data with repeated rows
- **WHEN** 20 列相同資料加 1 列離群值，`centers=2`、`NStart=1`，seed 1 到 50
- **THEN** 50 次全部成功

