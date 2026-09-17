# verification-integrity Specification

## Purpose
驗證要能被相信：跑不起來的檢查必須讓人看得見，不能靜默通過；CI 跑的工具版本必須釘住，否則同一份程式碼在不同時間會得到不同結論。

## Requirements

### Requirement: A check that could not run says so in a way that fails
The system SHALL provide a mode in which any verification skipped for a missing reference implementation fails the run instead, naming the missing tool and the verification that went unperformed.

#### Scenario: A reference toolchain is absent under strict mode

- **WHEN** the suite runs with reference-toolchain verification required and the reference implementation is absent
- **THEN** the affected check fails rather than skipping
- **AND** the failure names both the missing tool and what was left unverified

#### Scenario: A reference toolchain is absent by default

- **WHEN** the suite runs without reference-toolchain verification required and the reference implementation is absent
- **THEN** the affected check skips as before
- **AND** the run still passes, so the suite remains usable on a machine that has none of them

#### Scenario: A check is behind an opt-in flag

- **WHEN** a check is opt-in because its reference implementation is usually absent, and the suite runs with reference-toolchain verification required
- **THEN** that check runs rather than skipping, because the reason it was opt-in no longer holds

### Requirement: Every reference-toolchain gate is subject to the mode
The system SHALL route every check that depends on an external reference implementation through the same gate, so no such check can skip silently.

#### Scenario: A verification depends on R

- **WHEN** a check requires R with its analysis packages
- **THEN** its absence is reported through the shared gate

#### Scenario: A verification depends on Python

- **WHEN** a check requires Python with a scientific stack, scikit-learn, or an ONNX runtime
- **THEN** its absence is reported through the shared gate

### Requirement: Continuous integration provides the toolchains it gates on
The system SHALL install, in every workflow that exists to run a reference-implementation comparison, every dependency that comparison's own gate requires. Every opt-in comparison that strict mode turns on SHALL be run by a step of such a workflow.

#### Scenario: A parity workflow runs its suite

- **WHEN** a workflow exists to run a cross-language parity suite
- **THEN** the dependencies it installs satisfy that suite's gate
- **AND** the suite executes rather than skipping

#### Scenario: The full verification set runs in continuous integration

- **WHEN** continuous integration runs the reference-implementation verifications
- **THEN** it does so with reference-toolchain verification required
- **AND** a check that could not run fails the workflow

#### Scenario: An opt-in comparison has a step that runs it

- **WHEN** a test is opt-in because its reference implementation is usually absent, such as the portfolio comparison against cvxpy
- **THEN** the reference verification workflow installs that implementation and has a step whose `go test` pattern selects the test
- **AND** the test executes in that step rather than skipping

### Requirement: CI pins the versions it runs

每個 workflow 對同一個 action SHALL 使用同一個主版本；linter SHALL 釘在明確版號，SHALL NOT 使用 `latest`，否則 lint 結果無法重現。

#### Scenario: Two workflows using the same action
- **WHEN** 比較任兩個 workflow 的 `actions/checkout` 與 `actions/setup-go`
- **THEN** 版本相同

### Requirement: A factor solution is compared up to factor order and sign

因子分析的 R parity 比對 SHALL 先以載荷找出讓 `max|L_go − L_r|` 最小的欄位排列與正負號，再把同一組排列與正負號套用到所有以因子為索引的欄位（載荷、結構矩陣、`Phi`、分數共變異、分數、分數係數、旋轉矩陣、解釋比例與累積比例）之後才比較。SHALL NOT 把欄位換序或變號當成數值差異。

#### Scenario: R's rotation matrix predates its sign standardisation
- **WHEN** 兩邊的載荷在對齊後於容忍度內相同，而 R 的旋轉矩陣某一欄正負號相反
- **THEN** 旋轉矩陣的比對通過

#### Scenario: Factors in a different order
- **WHEN** 兩邊的載荷只差欄位順序
- **THEN** 載荷、`Phi` 與分數的比對都通過

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

### Requirement: A cached reference baseline is bound to the toolchain that produced it

參考基準的快取鍵 SHALL 包含產生它的工具鏈版本：`Rscript` 為 R 版本與 psych、GPArotation、jsonlite 的版本，`python` 為直譯器與 numpy、scipy、statsmodels 的版本。工具鏈版本改變時，舊快取 SHALL NOT 被當成現行基準使用。

#### Scenario: psych is upgraded
- **WHEN** 機器上的 psych 從 2.5.x 升到 2.6.5
- **THEN** 下一次執行重新產生所有 R 基準，不使用舊版本產生的快取

### Requirement: A reference baseline that draws random numbers is seeded

產生參考值的腳本若會呼叫使用亂數的參考實作（例如 `psych::fa` 的多個起點），SHALL 在呼叫前以固定的種子設定亂數產生器，使同一組輸入在任何工作階段都產生完全相同的輸出。快取的參考值 SHALL NOT 取決於建立快取時的工作階段亂數狀態。

#### Scenario: The same payload twice
- **WHEN** 以同一組因素分析輸入（`two_blocks`、ML 抽取、oblimin、迴歸分數、兩個因子）不經快取執行參考值腳本兩次
- **THEN** 兩次的輸出逐位元組相同

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
