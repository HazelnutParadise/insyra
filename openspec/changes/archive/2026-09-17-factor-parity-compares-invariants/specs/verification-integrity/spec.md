## MODIFIED Requirements

### Requirement: A multimodal rotation is judged by its criterion value

對梯度投影旋轉（varimax、quartimax、geomin、bentler、quartimin、oblimin、simplimax），比對 SHALL 先以對齊後載荷的最大差距判斷兩邊是否為同一個解。差距在 `factorRotationTol`（1e-2）之內為同一個解：因子座標下的欄位（載荷、結構矩陣、`Phi`、旋轉矩陣、解釋比例、分數、分數係數與分數共變異）SHALL 以 `factorRotationTol` 比對，不論載荷差距是否已在 `factorParityTol` 之內；此時我方準則值 SHALL NOT 高於 R 超過 1e-8 + 1e-4·|f_R|。差距超過 `factorRotationTol` 時，我方準則值低於 R 超過該範圍 SHALL 視為更好的解而通過，並記錄兩個準則值；否則 SHALL 失敗並指出兩個準則值。準則值相近 SHALL NOT 單獨作為同一個解的依據。Promax 沒有準則，SHALL 維持逐元素比對。

#### Scenario: A lower minimum than R
- **WHEN** simplimax 的起點找到比 R 的解準則值更低、載荷差距超過 `factorRotationTol` 的解
- **THEN** 因子座標下的欄位不比對，比對通過，並記錄兩個準則值

#### Scenario: The same minimum, reached to different precision
- **WHEN** 兩邊載荷差 3.3e-3、`Phi` 差 6.0e-3，準則值在範圍之內
- **THEN** 因子座標下的欄位以 `factorRotationTol` 比對並通過

#### Scenario: A worse minimum than R
- **WHEN** 我方解的準則值高於 R 超過 1e-8 + 1e-4·|f_R|，不論載荷差距多少
- **THEN** 載荷比對失敗，訊息含兩個準則值

#### Scenario: Loadings within the strict tolerance, Phi not
- **WHEN** 兩邊載荷差 9e-6、`Phi` 差 9e-5
- **THEN** `Phi` 以 `factorRotationTol` 比對並通過

#### Scenario: A different solution without a lower criterion
- **WHEN** 兩邊載荷差距超過 `factorRotationTol`，而我方準則值沒有低於 R
- **THEN** 載荷比對失敗，訊息含兩個準則值

## ADDED Requirements

### Requirement: Rotation-invariant quantities are compared strictly

因子分析的 R parity 比對 SHALL 對每一種旋轉、每一種判定結果（包括不同的極小值與 Promax），以 `factorParityTol` 比較 L·Phi·L' 與 S·L'，因為旋轉不會改變它們。迴歸分數與 Bartlett 分數 SHALL 另以 `scoreInvariantTol`（1e-4）比較 W·L'。我方的結構矩陣 SHALL 在 1e-10 之內等於載荷乘以 `Phi`。

#### Scenario: The same model at a different point of the minimum
- **WHEN** 兩邊 `Phi` 差 6.0e-3，而擬合的是同一個模型
- **THEN** L·Phi·L' 與 S·L' 在 `factorParityTol` 之內，比對通過

#### Scenario: A frame error
- **WHEN** `Phi` 的相關係數正負號錯誤，或結構矩陣沒有乘上 `Phi`
- **THEN** L·Phi·L' 或 S·L' 偏離超過 0.1，比對失敗

#### Scenario: A different lower minimum
- **WHEN** 我方找到準則值更低的另一個極小值
- **THEN** 因子座標下的欄位不比對，L·Phi·L' 與 S·L' 仍以 `factorParityTol` 比對
