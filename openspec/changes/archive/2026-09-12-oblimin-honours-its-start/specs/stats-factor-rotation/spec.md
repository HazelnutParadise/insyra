## ADDED Requirements

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
