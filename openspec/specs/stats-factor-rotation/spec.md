# stats-factor-rotation Specification

## Purpose
Factor rotation turns a fitted loading matrix into an equivalent one that is easier to read, and the whole point is that it is equivalent: the common part of the model must survive it untouched. This capability says what that means in numbers, what a multi-start search may start from so the guarantee holds, and what the convergence flag a caller reads is allowed to claim. It exists because none of the three was enforced — an orthogonal rotation could come back oblique, and a rotation that ran out of iterations reported success.
## Requirements
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

### Requirement: Every rotation runs from the start it is given

多起點搜尋中的每一個起點 SHALL 真的被當成該次旋轉的起點使用，任何旋轉方法 SHALL NOT 忽略傳入的起點而改用自己固定的起點。`Restarts` 對所有方法 SHALL 表示同一件事：從 `Restarts` 個不同的正交起點各跑一次，依既有規則挑出最佳解。

#### Scenario: Oblimin and quartimin agree on the same starts
- **WHEN** 以 `gamma = 0` 的 Oblimin 與 Quartimin 各自從同一份起點清單旋轉同一組載荷
- **THEN** 兩者回傳的準則值在 1e-12 之內相同，因為 `gamma = 0` 的 Oblimin 準則就是 Quartimin 準則

#### Scenario: A start other than the identity wins
- **WHEN** 在一組單位矩陣起點收斂到局部最小值的載荷上，以 `Restarts` 大於 1 旋轉
- **THEN** 回傳的解是所有收斂起點中準則值最小者，該準則值低於單位矩陣起點得到的準則值

#### Scenario: One start is still the identity alone
- **WHEN** `Restarts` 為 1 或更小
- **THEN** 每個方法的結果與多起點搜尋存在之前逐位元相同，包括 Oblimin

### Requirement: The default number of starts follows the reference implementation

`DefaultFactorAnalysisOptions()` SHALL 把 `Rotation.Restarts` 設為 20，與 psych 2.6.5 的 `n.rotations` 預設相同。呼叫端沒有設定（值為 0 或負數）時 SHALL 視為 20。呼叫端明確設 1 時 SHALL 只跑單位矩陣這一個起點。

#### Scenario: The defaults
- **WHEN** 取得 `DefaultFactorAnalysisOptions()`
- **THEN** `Rotation.Restarts` 為 20

#### Scenario: Restarts left unset
- **WHEN** 以 `Rotation.Restarts` 為 0 呼叫 `FactorAnalysis`
- **THEN** 旋轉從 20 個起點各跑一次

#### Scenario: One start on request
- **WHEN** 以 `Rotation.Restarts` 為 1 呼叫 `FactorAnalysis`
- **THEN** 唯一的起點是單位矩陣，結果與多起點搜尋存在之前逐位元相同

### Requirement: The informed start is built at the search's own tolerance

建立起點清單時，用 Varimax 產生的資訊起點 SHALL 以該次旋轉的 `eps` 與 `maxIter` 執行，SHALL NOT 使用比旋轉本身更嚴的容忍度或更高的迭代上限。起點只需要是正交矩陣，正交性另有驗證。

#### Scenario: Building twenty starts
- **WHEN** 以旋轉的 `eps = 1e-5`、`maxIter = 1000` 建立 20 個起點
- **THEN** 資訊起點以同樣的 `eps` 與 `maxIter` 計算，並通過正交性驗證

### Requirement: Non-convergence is reported once per search

多起點搜尋 SHALL 只在被選中的解沒有收斂時記錄一次警告，警告 SHALL 指出方法名稱、起點數與迭代上限。個別起點跑到迭代上限 SHALL NOT 記錄警告，也 SHALL NOT 寫入全域錯誤緩衝區；它只在 debug 層級可見。

#### Scenario: Every start fails to converge
- **WHEN** 以 5 個起點旋轉，且每個起點都跑到迭代上限
- **THEN** 恰好記錄一則警告，全域錯誤緩衝區恰好多一筆，`RotationConverged` 為 false

#### Scenario: The chosen solution converged
- **WHEN** 以 20 個起點旋轉，且被選中的解收斂
- **THEN** 不記錄任何警告，全域錯誤緩衝區不增加，不論其他起點是否收斂

