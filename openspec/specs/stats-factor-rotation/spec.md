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
- **WHEN** 以旋轉的 `eps = 1e-5`、`maxIter = 2000` 建立 20 個起點
- **THEN** 資訊起點以同樣的 `eps` 與 `maxIter` 計算，並通過正交性驗證

### Requirement: Non-convergence is reported once per search

多起點搜尋 SHALL 只在被選中的解沒有收斂時記錄一次警告，警告 SHALL 指出方法名稱、起點數與迭代上限。個別起點跑到迭代上限 SHALL NOT 記錄警告，也 SHALL NOT 寫入全域錯誤緩衝區；它只在 debug 層級可見。

#### Scenario: Every start fails to converge
- **WHEN** 以 5 個起點旋轉，且每個起點都跑到迭代上限
- **THEN** 恰好記錄一則警告，全域錯誤緩衝區恰好多一筆，`RotationConverged` 為 false

#### Scenario: The chosen solution converged
- **WHEN** 以 20 個起點旋轉，且被選中的解收斂
- **THEN** 不記錄任何警告，全域錯誤緩衝區不增加，不論其他起點是否收斂

### Requirement: Promax pre-rotates the way psych does

Promax SHALL 先以梯度投影的 Varimax 從單位矩陣旋轉（不做 Kaiser 正規化，`eps = 1e-5`），再進行目標矩陣的最小平方步驟，與 psych 2.6.5 的 `Promax()` 相同。SHALL NOT 使用 `stats::varimax` 的成對旋轉演算法作為前置旋轉，因為在 varimax 準則接近平坦的資料上兩者停在不同角度，而 Promax 會把該差距放大到四次方。

#### Scenario: The parity suite's ten-row table
- **WHEN** 以 psych 2.6.5 相同的 Kaiser 加權載荷呼叫 `Promax(weighted, 4)`
- **THEN** 載荷與 psych 的 `Promax(weighted, m = 4)` 在 5e-4 之內相同，`Phi[0,1]` 亦然

#### Scenario: Data with a clear varimax optimum
- **WHEN** 資料的 varimax 準則有明確的最佳解
- **THEN** 前置旋轉的更換只在收斂容忍度的量級上改變結果

### Requirement: The random starts are the same on every platform

多起點搜尋的隨機起點 SHALL 由固定的種子產生，SHALL NOT 取決於載荷的浮點位元。抽取的結果在不同架構上只能重現到浮點誤差的量級，同一張表的 ML 載荷在 amd64 與 arm64 上第八位小數就不同。用這些位元雜湊出來的種子，會在兩個架構上抽出毫不相干的起點，同一個呼叫就會在不同平台落到不同的解。

#### Scenario: Loadings that differ only in the last digits
- **WHEN** 兩組載荷只有一個元素相差 2e-8（同一張表在 amd64 與 arm64 上 ML 抽取結果的差距），以相同的 `Restarts` 建立起點清單
- **THEN** 兩份清單的隨機起點逐位元相同
- **AND** 以 `Restarts` 5 旋轉的準則值落在同一個盆地

#### Scenario: The same call on different platforms
- **WHEN** 在 amd64 與 arm64 上以相同資料與選項呼叫 `FactorAnalysis`，且 `Rotation.Restarts` 大於 1
- **THEN** 兩者使用相同的隨機起點，選中的解落在同一個盆地

### Requirement: Gradient projection rotations step the way the reference does

梯度投影旋轉（Varimax、Quartimax、GeominT、BentlerT、Quartimin、Oblimin、GeominQ、BentlerQ、Simplimax，以及多起點搜尋的資訊起點和 Promax 的前置 Varimax）SHALL 採用 GPArotation 2026.8.2 的預設演算法逐步計算。第一次迭代把步長加倍，之後的步長依前一次迭代中旋轉矩陣與投影梯度的變化估算，並限制在 1e-10 到 20 之間。一步嘗試的準則值只要比最近 10 次迭代中最大的準則值小得夠多，就 SHALL 被接受，否則步長減半，最多嘗試 11 次。呼叫端沒有指定時，SHALL 使用 `eps = 1e-5` 與最多 2000 次迭代。斜交旋轉矩陣無法直接求逆時，SHALL 以奇異值分解求擬反矩陣繼續，SHALL NOT 略過該次嘗試。

#### Scenario: The same loadings and start as GPArotation
- **WHEN** 以與 GPArotation 2026.8.2 相同的載荷、起點、準則、`eps` 與迭代上限旋轉
- **THEN** 前 6 次迭代的準則值與步長，和 GPArotation 的相對誤差都在 1e-10 之內
- **AND** 收斂與否相同，最終準則值的相對誤差在 1e-8 之內，載荷在 1e-4 之內

#### Scenario: A nearly flat criterion
- **WHEN** 以兩個因子旋轉 parity 套件 `two_blocks`（MINRES）與 `missing_rows`（PCA）十列表的未旋轉載荷，從單位矩陣出發，以 Quartimin 或 GeominQ 旋轉
- **THEN** 旋轉回報收斂，準則值與 GPArotation 2026.8.2 相同

#### Scenario: Iteration cap left unset
- **WHEN** 呼叫端沒有指定旋轉的迭代上限，且沒有任何起點收斂
- **THEN** 每個起點最多迭代 2000 次，未收斂的警告指出上限是 2000
