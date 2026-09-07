# interpolation-correctness Specification

## Purpose
插值法必須滿足它名稱所宣稱的條件。

## Requirements
### Requirement: Hermite interpolation satisfies its conditions

`HermiteInterpolation(data, derivatives, x)` SHALL 在每個節點 i 通過 `data[i]`，且其導數 SHALL 等於 `derivatives[i]`；對次數足夠低的多項式 SHALL 精確重現。

#### Scenario: Two-node cubic
- **WHEN** `data=[0,0]`、`derivatives=[1,0]`，求 x=0.5
- **THEN** 結果為 0.125（唯一滿足條件的三次多項式 t(1-t)²）

