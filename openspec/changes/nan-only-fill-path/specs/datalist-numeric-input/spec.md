## ADDED Requirements

### Requirement: A NaN-only mean fill has a replacement that keeps its behaviour

`FillNaNWithMean`'s Deprecated notice SHALL name `dl.ReplaceNaNsWith(dl.Clone().ClearNilsAndNaNs().Mean())` as the replacement that fills `NaN` alone, instead of only `FillWithMean`, which also fills `nil`. On a list of numbers, `nil` and `NaN`, the replacement SHALL give the same values in every cell as `FillNaNWithMean`, leaving `nil` cells as they are.

#### Scenario: The replacement matches
- **WHEN** 對 `[1, nil, NaN, 3, NaN]` 分別呼叫 `FillNaNWithMean()` 與 `ReplaceNaNsWith(Clone().ClearNilsAndNaNs().Mean())`
- **THEN** 兩者每一格的數值相同：`[1, nil, 2, 3, 2]`，nil 保留
