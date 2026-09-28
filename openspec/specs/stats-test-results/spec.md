# stats-test-results Specification

## Purpose
The part every hypothesis test result shares, exported so one function or one slice can take any test's result, and how a result field says that it does not apply to the test that produced it.

## Requirements

### Requirement: Every hypothesis test result exposes its shared part

`stats` SHALL 匯出 `TestResult` 型別，欄位為 `Statistic float64`、`PValue float64`、`DF *float64`、`CI *[2]float64` 與 `EffectSizes []EffectSizeEntry`。`TTestResult`、`ZTestResult`、`FTestResult`、`ChiSquareTestResult`、`CorrelationResult`、`WilcoxonTestResult`、`MannWhitneyUResult`、`KruskalWallisResult` 與 `FriedmanTestResult` SHALL 以名稱 `TestResult` 內嵌它，讓 `r.PValue` 這類欄位存取維持不變。

`stats` SHALL 匯出介面 `HypothesisTestResult`，只有一個方法 `Base() *TestResult`。`*TestResult` SHALL 實作這個方法並回傳自己，因此上列每個型別的指標 SHALL 透過內嵌滿足這個介面，`Base()` SHALL 回傳該結果內嵌的那一份 `TestResult`，而不是複本。

#### Scenario: One function over any test
- **WHEN** 把 `SingleSampleTTest`、`MannWhitneyU` 與 `ChiSquareIndependenceTest` 的結果放進同一個 `[]stats.HypothesisTestResult`，逐一讀取 `r.Base().PValue`
- **THEN** 讀到的值與各結果的 `PValue` 欄位相同

#### Scenario: Base points at the embedded value
- **WHEN** 對 `TwoSampleTTest` 的結果 `res` 呼叫 `res.Base()`
- **THEN** 回傳的指標等於 `&res.TestResult`

#### Scenario: Every result type satisfies the interface
- **WHEN** 編譯 `var _ stats.HypothesisTestResult = (*stats.TTestResult)(nil)`，以及上列其餘八個型別的同樣宣告
- **THEN** 全部編譯通過

### Requirement: A result field that does not apply is nil

內嵌 `TestResult` 的結果型別中，只有部分呼叫會填入的欄位 SHALL 是指標，不適用時 SHALL 為 nil；每次呼叫都會填入的欄位 SHALL 是一般值。SHALL NOT 以 `NaN` 或 0 表示欄位不適用。具體而言：

- `TTestResult.Mean` 與 `ZTestResult.Mean` SHALL 都是 `float64`。`PairedTTest` SHALL 以 `data1` 與 `data2` 的平均數填入 `Mean` 與 `Mean2`，以成對數填入 `N` 與 `N2`，並照舊填入 `MeanDiff`。`SingleSampleTTest` 的 `Mean2`、`N2`、`MeanDiff` SHALL 為 nil；`TwoSampleTTest` 的 `MeanDiff` SHALL 為 nil。
- `WilcoxonTestResult.Z` 與 `MannWhitneyUResult.Z` SHALL 是 `*float64`。走常態近似（`Method` 為 `"asymptotic"`）時 SHALL 指向標準化 z 值，包括所有值同分、z 無法定義而為 `NaN` 的情況；沒有計算 z 時，即走精確分布，或 Wilcoxon 的差值全為零而 `Method` 為 `"undefined"`，SHALL 為 nil。
- `FTestResult.DF2` SHALL 是 `*float64`；`BartlettTest` 的 `DF2` SHALL 為 nil，其他 F 檢定 SHALL 指向第二個自由度。

各檢定回報的數值 SHALL 與變更前相同。

#### Scenario: Paired t-test reports both means
- **WHEN** 呼叫 `PairedTTest(x, y)`，`x` 為 `52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8`，`y` 為 `48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5`
- **THEN** `Mean` 為 `x` 的平均數，`*Mean2` 為 `y` 的平均數，`*N2` 為 7，`*MeanDiff` 與變更前相同

#### Scenario: Exact Wilcoxon has no z
- **WHEN** `SingleSampleWilcoxon` 在沒有同分、`n` 不超過 50 的樣本上走精確分布
- **THEN** `Method` 為 `"exact"`，`Z` 為 nil

#### Scenario: Bartlett has no second degrees of freedom
- **WHEN** 呼叫 `BartlettTest([]insyra.IDataList{a, b, c})`
- **THEN** `*DF` 為 2，`DF2` 為 nil；`LeveneTest` 對同樣三組的 `DF2` 不是 nil
