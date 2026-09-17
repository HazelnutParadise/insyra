## ADDED Requirements

### Requirement: Simplimax minimises GPArotation's criterion

Simplimax 旋轉 SHALL 最小化 GPArotation 2026.8.2 `simplimax` 定義的準則：把平方載荷由小到大排序，恰好取最小的 `k` 個相加。沒有指定 `k` 時，`k` SHALL 等於變數數乘以（因子數減一）。多個平方載荷相等時，SHALL 依欄優先的位置順序決定取哪幾個（先比欄，同一欄再比列），SHALL NOT 把與第 `k` 小相等的其他值一併算入。旋轉本身、多起點搜尋挑選起點時，以及判斷兩個解是否為同一個極小值時，SHALL 使用同一個準則。

#### Scenario: Three factors
- **WHEN** 計算三個因子的載荷的 Simplimax 準則，沒有指定 `k`
- **THEN** `k` 等於變數數的兩倍，準則值與梯度與 GPArotation 2026.8.2 相同

#### Scenario: Equal squared loadings
- **WHEN** 有好幾個平方載荷與第 `k` 小的值相等
- **THEN** 準則恰好包含 `k` 個平方載荷，取欄優先位置在前的那些，準則值與梯度與 GPArotation 2026.8.2 相同

#### Scenario: Two factors without ties
- **WHEN** 旋轉兩個因子、沒有相等平方載荷的載荷
- **THEN** 準則值與梯度和這項要求之前相同
