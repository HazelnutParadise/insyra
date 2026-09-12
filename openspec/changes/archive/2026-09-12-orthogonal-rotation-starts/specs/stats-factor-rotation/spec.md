## ADDED Requirements

### Requirement: A rotation preserves the fitted model

任何旋轉 SHALL 保持共同因子重建出來的共變結構。正交旋轉的載荷 `L` 與未旋轉載荷 `Lu` SHALL 滿足 `L·L' = Lu·Lu'`，斜交旋轉 SHALL 滿足 `L·Φ·L' = Lu·Lu'`，兩者皆在浮點誤差之內。不滿足的解 SHALL NOT 被回傳，因為它描述的已經不是被配適的那個模型。

#### Scenario: An orthogonal rotation with more than one start
- **WHEN** 以 Varimax、Quartimax、BentlerT 或 GeominT 旋轉，且 `Restarts` 大於 1
- **THEN** `max|L·L' − Lu·Lu'|` 在 1e-10 之內
- **AND** 回傳的旋轉矩陣 `R` 滿足 `max|R'R − I|` 在 1e-10 之內

#### Scenario: An oblique rotation with more than one start
- **WHEN** 以 Quartimin、Oblimin、BentlerQ、GeominQ 或 Simplimax 旋轉，且 `Restarts` 大於 1
- **THEN** `max|L·Φ·L' − Lu·Lu'|` 在 1e-10 之內

#### Scenario: The default path
- **WHEN** `Restarts` 為 1 或更小
- **THEN** 唯一的起點是單位矩陣，多起點搜尋不介入結果

### Requirement: Every rotation start lies on the criterion's own manifold

多起點搜尋的每一個起點 SHALL 是正交矩陣。系統 SHALL 在使用前驗證起點的正交性，驗不過的起點 SHALL 被略過而不是照用。系統 SHALL NOT 把斜交方法（Promax、Target）產生的矩陣當成起點，因為它們不在正交群上，而梯度投影演算法只保證「相對於起點」的可行性。

#### Scenario: A start that is not orthogonal
- **WHEN** 建立起點清單
- **THEN** 每一個起點都滿足 `max|S'S − I|` 在 1e-8 之內

#### Scenario: A helper that fails while producing a start
- **WHEN** 用來產生起點的旋轉失敗，或交出不在正交群上的矩陣
- **THEN** 該起點被略過，並以隨機正交起點補上，起點總數不減少

### Requirement: Restarts is the number of starts

`Restarts` SHALL 等於實際嘗試的起點數量。系統 SHALL NOT 在使用者要求的數量之外無條件追加起點。

#### Scenario: Two restarts
- **WHEN** `Restarts` 為 2
- **THEN** 恰好嘗試 2 個起點

### Requirement: The convergence flag reports what happened

系統 SHALL 回報被選中的那一次旋轉是否收斂。在多個起點之間挑選時，SHALL 優先選擇收斂的解中準則值最小者；若全部都沒有收斂，SHALL 回傳準則值最小者並回報未收斂。系統 SHALL NOT 在缺少收斂資訊時預設回報已收斂。

#### Scenario: A rotation that does not converge
- **WHEN** 以極小的 `MaxIter` 與極嚴格的 `Eps` 旋轉，使其無法收斂
- **THEN** 回報未收斂，該結果一路傳到 `FactorAnalysisResult.RotationConverged`

#### Scenario: A rotation that converges
- **WHEN** 以一般的 `MaxIter` 在收斂得了的資料上旋轉
- **THEN** 回報已收斂

#### Scenario: One start converges and another does not
- **WHEN** 多個起點中只有一部分收斂
- **THEN** 被選中的是收斂那一群裡準則值最小的解
