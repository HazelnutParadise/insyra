## ADDED Requirements

### Requirement: Non-convergence is reported once per search

多起點搜尋 SHALL 只在被選中的解沒有收斂時記錄一次警告，警告 SHALL 指出方法名稱、起點數與迭代上限。個別起點跑到迭代上限 SHALL NOT 記錄警告，也 SHALL NOT 寫入全域錯誤緩衝區；它只在 debug 層級可見。

#### Scenario: Every start fails to converge
- **WHEN** 以 5 個起點旋轉，且每個起點都跑到迭代上限
- **THEN** 恰好記錄一則警告，全域錯誤緩衝區恰好多一筆，`RotationConverged` 為 false

#### Scenario: The chosen solution converged
- **WHEN** 以 20 個起點旋轉，且被選中的解收斂
- **THEN** 不記錄任何警告，全域錯誤緩衝區不增加，不論其他起點是否收斂
