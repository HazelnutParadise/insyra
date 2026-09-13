## ADDED Requirements

### Requirement: The random starts are the same on every platform

多起點搜尋的隨機起點 SHALL 由固定的種子產生，SHALL NOT 取決於載荷的浮點位元。抽取的結果在不同架構上只能重現到浮點誤差的量級，同一張表的 ML 載荷在 amd64 與 arm64 上第八位小數就不同。用這些位元雜湊出來的種子，會在兩個架構上抽出毫不相干的起點，同一個呼叫就會在不同平台落到不同的解。

#### Scenario: Loadings that differ only in the last digits
- **WHEN** 兩組載荷只有一個元素相差 2e-8（同一張表在 amd64 與 arm64 上 ML 抽取結果的差距），以相同的 `Restarts` 建立起點清單
- **THEN** 兩份清單的隨機起點逐位元相同
- **AND** 以 `Restarts` 5 旋轉的準則值落在同一個盆地

#### Scenario: The same call on different platforms
- **WHEN** 在 amd64 與 arm64 上以相同資料與選項呼叫 `FactorAnalysis`，且 `Rotation.Restarts` 大於 1
- **THEN** 兩者使用相同的隨機起點，選中的解落在同一個盆地
