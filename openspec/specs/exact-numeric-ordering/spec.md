# exact-numeric-ordering Specification

## Purpose
排序與排名的數值精度：整數以整數精度比較，不因 float64 而失真。

## Requirements
### Requirement: Integers compare exactly

比較兩個整數值 SHALL 以整數精度進行，SHALL NOT 經由 float64；包含混合有號／無號、以及超出 `int64` 範圍的 `uint64`。只有其中一方為浮點數時才 SHALL 以 float64 比較。

#### Scenario: Two int64 above 2^53
- **WHEN** 比較 `int64(2^53+1)` 與 `int64(2^53)`
- **THEN** 前者大於後者

### Requirement: Sorting and ranking keep that precision

`DataList.Sort`、`Rank` 與其他以 `CompareAny` 排序的路徑 SHALL 對大整數給出正確順序；`Rank` SHALL 以原始儲存格判定並列，SHALL NOT 以 float64 副本判定。

#### Scenario: Rank of three adjacent large integers
- **WHEN** 對 `[2^53+1, 2^53, 2^53+2]` 呼叫 `Rank()`
- **THEN** 結果為 `[2, 1, 3]`

