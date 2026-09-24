# interpolation-correctness Specification

## Purpose
插值法必須滿足它名稱所宣稱的條件：`HermiteInterpolation` 通過每個節點的值、符合每個給定的導數，並精確重現低次多項式。

## Requirements

### Requirement: Hermite interpolation satisfies its conditions

`HermiteInterpolation(data, derivatives, x)` SHALL 在每個節點 i 通過 `data[i]`，且其導數 SHALL 等於 `derivatives[i]`；對次數足夠低的多項式 SHALL 精確重現。

#### Scenario: Two-node cubic
- **WHEN** `data=[0,0]`、`derivatives=[1,0]`，求 x=0.5
- **THEN** 結果為 0.125（唯一滿足條件的三次多項式 t(1-t)²）

### Requirement: Every interpolation refuses a NaN

所有插值函式對 NaN 的 x SHALL 回傳 `ErrOutOfBounds`，SHALL NOT 回傳看起來合理的數值或 NaN。與 NaN 的比較永遠為 false，因此沒有防護的搜尋會停在第一個索引。

#### Scenario: A NaN x
- **WHEN** 以 NaN 呼叫 `NearestNeighborInterpolation`、`LagrangeInterpolation` 或 `NewtonInterpolation`
- **THEN** 回傳 `ErrOutOfBounds`，與 `Linear`／`Quadratic` 一致
