# exact-numeric-ordering Specification

## Purpose
排序路徑的數值精度：整數以整數精度比較，`DataList.Sort`、`DataTable.SortBy` 與 `Pivot` 的欄序不因經由 float64 而把相鄰的大整數視為相等。

## Requirements
### Requirement: Integers compare exactly

比較兩個整數值 SHALL 以整數精度進行，SHALL NOT 經由 float64；包含混合有號／無號、以及超出 `int64` 範圍的 `uint64`。只有其中一方為浮點數時才 SHALL 以 float64 比較。

#### Scenario: Two int64 above 2^53
- **WHEN** 比較 `int64(2^53+1)` 與 `int64(2^53)`
- **THEN** 前者大於後者

### Requirement: Sorting keeps that precision

`DataList.Sort`、`DataTable.SortBy`、`Pivot` 的欄序與其他以 `CompareAny` 排序的路徑 SHALL 對大整數給出正確順序。

#### Scenario: Sort of three adjacent large integers
- **WHEN** 對 `[2^53+1, 2^53, 2^53+2]` 呼叫 `Sort()`
- **THEN** 結果為 `[2^53, 2^53+1, 2^53+2]`
