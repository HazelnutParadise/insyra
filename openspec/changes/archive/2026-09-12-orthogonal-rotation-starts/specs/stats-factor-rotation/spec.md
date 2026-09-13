## ADDED Requirements

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
