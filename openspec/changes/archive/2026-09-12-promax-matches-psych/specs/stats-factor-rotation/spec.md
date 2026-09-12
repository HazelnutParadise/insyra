## ADDED Requirements

### Requirement: Promax pre-rotates the way psych does

Promax SHALL 先以梯度投影的 Varimax 從單位矩陣旋轉（不做 Kaiser 正規化，`eps = 1e-5`），再進行目標矩陣的最小平方步驟，與 psych 2.6.5 的 `Promax()` 相同。SHALL NOT 使用 `stats::varimax` 的成對旋轉演算法作為前置旋轉，因為在 varimax 準則接近平坦的資料上兩者停在不同角度，而 Promax 會把該差距放大到四次方。

#### Scenario: The parity suite's ten-row table
- **WHEN** 以 psych 2.6.5 相同的 Kaiser 加權載荷呼叫 `Promax(weighted, 4)`
- **THEN** 載荷與 psych 的 `Promax(weighted, m = 4)` 在 5e-4 之內相同，`Phi[0,1]` 亦然

#### Scenario: Data with a clear varimax optimum
- **WHEN** 資料的 varimax 準則有明確的最佳解
- **THEN** 前置旋轉的更換只在收斂容忍度的量級上改變結果
