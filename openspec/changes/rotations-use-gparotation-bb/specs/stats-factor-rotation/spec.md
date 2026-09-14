## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: The informed start is built at the search's own tolerance

建立起點清單時，用 Varimax 產生的資訊起點 SHALL 以該次旋轉的 `eps` 與 `maxIter` 執行，SHALL NOT 使用比旋轉本身更嚴的容忍度或更高的迭代上限。起點只需要是正交矩陣，正交性另有驗證。

#### Scenario: Building twenty starts
- **WHEN** 以旋轉的 `eps = 1e-5`、`maxIter = 2000` 建立 20 個起點
- **THEN** 資訊起點以同樣的 `eps` 與 `maxIter` 計算，並通過正交性驗證
