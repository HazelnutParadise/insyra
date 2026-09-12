## ADDED Requirements

### Requirement: A factor solution is compared up to factor order and sign

因子分析的 R parity 比對 SHALL 先以載荷找出讓 `max|L_go − L_r|` 最小的欄位排列與正負號，再把同一組排列與正負號套用到所有以因子為索引的欄位（載荷、結構矩陣、`Phi`、分數共變異、分數、分數係數、旋轉矩陣、解釋比例與累積比例）之後才比較。SHALL NOT 把欄位換序或變號當成數值差異。

#### Scenario: R's rotation matrix predates its sign standardisation
- **WHEN** 兩邊的載荷在對齊後於容忍度內相同，而 R 的旋轉矩陣某一欄正負號相反
- **THEN** 旋轉矩陣的比對通過

#### Scenario: Factors in a different order
- **WHEN** 兩邊的載荷只差欄位順序
- **THEN** 載荷、`Phi` 與分數的比對都通過

### Requirement: A multimodal rotation is judged by its criterion value

對梯度投影旋轉（varimax、quartimax、geomin、bentler、quartimin、oblimin、simplimax），對齊後的載荷若仍超出容忍度，比對 SHALL 以該方法自己的準則函數分別評估兩邊的載荷。兩邊準則值在相對 1e-4 之內 SHALL 視為同一個最小值，因子座標下的欄位改以準則曲率所能決定的精度（`factorRotationTol`）比對；我方準則值更低的 SHALL 視為更好的解而通過，並記錄兩個準則值；高於 R 超過該範圍的 SHALL 失敗並指出兩個準則值。Promax 沒有準則，SHALL 維持逐元素比對。

#### Scenario: A lower minimum than R
- **WHEN** simplimax 的 20 個起點找到準則值 0.1445 的解，而 R 回傳 0.1599 的解
- **THEN** 比對通過，並記錄兩個準則值

#### Scenario: The same minimum, reached to different precision
- **WHEN** 兩邊準則值相差在相對 1e-4 之內，而載荷相差 1e-4
- **THEN** 因子座標下的欄位以 `factorRotationTol` 比對並通過，並記錄兩個準則值

#### Scenario: A worse minimum than R
- **WHEN** 我方解的準則值高於 R 的解超過相對 1e-4
- **THEN** 載荷比對失敗，訊息含兩個準則值

### Requirement: A cached reference baseline is bound to the toolchain that produced it

參考基準的快取鍵 SHALL 包含產生它的工具鏈版本：`Rscript` 為 R 版本與 psych、GPArotation、jsonlite 的版本，`python` 為直譯器與 numpy、scipy、statsmodels 的版本。工具鏈版本改變時，舊快取 SHALL NOT 被當成現行基準使用。

#### Scenario: psych is upgraded
- **WHEN** 機器上的 psych 從 2.5.x 升到 2.6.5
- **THEN** 下一次執行重新產生所有 R 基準，不使用舊版本產生的快取
