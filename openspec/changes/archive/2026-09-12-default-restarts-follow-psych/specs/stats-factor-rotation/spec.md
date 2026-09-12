## ADDED Requirements

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
