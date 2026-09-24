## ADDED Requirements

### Requirement: Every interpolation refuses a NaN

所有插值函式對 NaN 的 x SHALL 回傳 `ErrOutOfBounds`，SHALL NOT 回傳看起來合理的數值或 NaN。與 NaN 的比較永遠為 false，因此沒有防護的搜尋會停在第一個索引。

#### Scenario: A NaN x
- **WHEN** 以 NaN 呼叫 `NearestNeighborInterpolation`、`LagrangeInterpolation` 或 `NewtonInterpolation`
- **THEN** 回傳 `ErrOutOfBounds`，與 `Linear`／`Quadratic` 一致
