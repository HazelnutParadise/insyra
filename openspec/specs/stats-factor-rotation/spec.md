# stats-factor-rotation Specification

## Purpose
Factor rotation turns a fitted loading matrix into an equivalent one that is easier to read. On this line the capability records what the convergence flag a caller reads is allowed to claim: it describes the rotation that was actually returned, because it used to report success for a rotation that ran out of iterations.

## Requirements
### Requirement: The convergence flag reports what happened

系統 SHALL 回報實際回傳的那一次旋轉是否收斂，並 SHALL 一路傳到 `FactorAnalysisResult.RotationConverged`。系統 SHALL NOT 因為被選中的候選解沒有帶收斂旗標，就預設回報已收斂。Promax 是封閉式旋轉、沒有迭代，回報已收斂。

#### Scenario: A rotation that does not converge
- **WHEN** 以極小的 `MaxIter` 與極嚴格的 `Eps` 旋轉，使其無法收斂
- **THEN** 回報未收斂，該結果一路傳到 `FactorAnalysisResult.RotationConverged`

#### Scenario: A rotation that converges
- **WHEN** 以一般的 `MaxIter` 在收斂得了的資料上旋轉
- **THEN** 回報已收斂

#### Scenario: More than one start
- **WHEN** `Restarts` 大於 1
- **THEN** 回報的是依準則值選中的那個解本身的收斂狀態

### Requirement: Non-convergence is reported once per search

多起點搜尋 SHALL 只在被選中的解沒有收斂時記錄一次警告，警告 SHALL 指出方法名稱、起點數與迭代上限。個別起點跑到迭代上限 SHALL NOT 記錄警告，也 SHALL NOT 寫入全域錯誤緩衝區；它只在 debug 層級可見。

#### Scenario: Every start fails to converge
- **WHEN** 以 5 個起點旋轉，且每個起點都跑到迭代上限
- **THEN** 恰好記錄一則警告，全域錯誤緩衝區恰好多一筆，`RotationConverged` 為 false

#### Scenario: The chosen solution converged
- **WHEN** 以 20 個起點旋轉，且被選中的解收斂
- **THEN** 不記錄任何警告，全域錯誤緩衝區不增加，不論其他起點是否收斂
