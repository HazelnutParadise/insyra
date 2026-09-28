## ADDED Requirements

### Requirement: Hypothesis tests take their alternative and confidence level in an optional options struct

`SingleSampleTTest`、`TwoSampleTTest`、`PairedTTest` SHALL 以最後一個參數 `opts ...TTestOptions` 接收設定；`SingleSampleZTest`、`TwoSampleZTest` SHALL 以 `opts ...ZTestOptions` 接收；`SingleSampleWilcoxon`、`PairedWilcoxon` SHALL 以 `opts ...WilcoxonOptions` 接收；`MannWhitneyU` SHALL 以 `opts ...MannWhitneyUOptions` 接收。這四個型別 SHALL 各有 `Alternative AlternativeHypothesis` 與 `ConfidenceLevel float64` 兩個欄位。

`Alternative` 為空字串時 SHALL 視為 `TwoSided`；為 `TwoSided`、`Greater` 或 `Less` 以外的值時 SHALL 回傳錯誤 `alternative must be two-sided, greater or less, got "<值>"`。`ConfidenceLevel` 為 0 時 SHALL 視為 0.95；嚴格介於 0 與 1 之間時 SHALL 照用；其他值，包括 `NaN`、負數與大於等於 1 的值，SHALL 回傳錯誤 `confidence level must be strictly between 0 and 1, got <值>`。給了一個以上的設定值時 SHALL 回傳錯誤 `at most one stats.<型別> may be given, got <個數>`。這些檢查 SHALL 在讀取任何樣本之前完成，出錯時 SHALL NOT 回傳結果。

`mu`、`sigma`、`sigma1`、`sigma2` 與 `TwoSampleTTest` 的 `equalVariance` SHALL 維持為必填的位置參數。

#### Scenario: No options means two-sided at 95%
- **WHEN** 呼叫 `SingleSampleTTest(x, 50)`，`x` 為 `52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8`
- **THEN** 結果與 `SingleSampleTTest(x, 50, TTestOptions{Alternative: TwoSided, ConfidenceLevel: 0.95})` 逐位元相同，p 值為 0.2973295155（R `t.test(x, mu = 50)`）

#### Scenario: NaN as a level is refused
- **WHEN** 呼叫 `SingleSampleWilcoxon(x, 50, WilcoxonOptions{ConfidenceLevel: math.NaN()})`
- **THEN** 回傳錯誤 `confidence level must be strictly between 0 and 1, got NaN`，結果為 nil

#### Scenario: Empty alternative in a rank test
- **WHEN** 呼叫 `MannWhitneyU(x, y, MannWhitneyUOptions{})`
- **THEN** 以雙尾檢定計算，與 `MannWhitneyU(x, y, MannWhitneyUOptions{Alternative: TwoSided})` 結果相同，不回傳錯誤

#### Scenario: Two options values
- **WHEN** 呼叫 `SingleSampleZTest(x, 50, 10, ZTestOptions{}, ZTestOptions{})`
- **THEN** 回傳錯誤 `at most one stats.ZTestOptions may be given, got 2`

#### Scenario: Unknown alternative
- **WHEN** 呼叫 `PairedTTest(x, y, TTestOptions{Alternative: "sideways"})`
- **THEN** 回傳錯誤 `alternative must be two-sided, greater or less, got "sideways"`

### Requirement: The t-tests test one-sided alternatives as R's t.test does

`SingleSampleTTest`、`TwoSampleTTest` 與 `PairedTTest` 在 `Alternative` 為 `Greater` 或 `Less` 時，SHALL 回傳與 R `t.test(..., alternative = "greater" | "less", conf.level = cl)` 相同的統計量、自由度、p 值與單尾信賴界限：`Greater` 的區間 SHALL 為 `[下界, +Inf]`，`Less` 的區間 SHALL 為 `[-Inf, 上界]`。統計量、自由度與效果量 SHALL 與 `Alternative` 無關。樣本沒有變異而統計量為 `+Inf` 或 `-Inf`、且自由度為有限正數時，p 值 SHALL 依 `Alternative` 決定為 0 或 1；統計量或自由度為 `NaN` 時（例如兩組都沒有變異的 Welch 檢定），p 值 SHALL 為 `NaN`。`Alternative` 為 `TwoSided` 時的結果 SHALL 與變更前逐位元相同。

#### Scenario: One-sample, greater
- **WHEN** 呼叫 `SingleSampleTTest(x, 50, TTestOptions{Alternative: Greater})`，`x` 同上
- **THEN** 統計量為 1.141068052，p 值為 0.1486647578，區間為 `[48.00161745, +Inf]`，與 R `t.test(x, mu = 50, alternative = "greater")` 相差不超過 1e-8

#### Scenario: Paired, less, at 90%
- **WHEN** 呼叫 `PairedTTest(x, y, TTestOptions{Alternative: Less, ConfidenceLevel: 0.9})`，`y` 為 `48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5`
- **THEN** 統計量為 6.0869715254，p 值為 0.9995528678，區間為 `[-Inf, 3.6389332568]`

#### Scenario: Constant data against the direction of the effect
- **WHEN** 呼叫 `SingleSampleTTest(NewDataList(5.0, 5.0, 5.0), 4, TTestOptions{Alternative: Less})`
- **THEN** 統計量為 `+Inf`，p 值為 1，錯誤為 nil

#### Scenario: Cross-language check
- **WHEN** 參考驗證工作流程執行 t 檢定的跨語言測試
- **THEN** 三種 t 檢定在 `TwoSided`、`Greater`、`Less` 與至少兩種信賴水準下，統計量、自由度、p 值與區間都與 R `t.test` 及 SciPy 的對應函式相符

### Requirement: The k-sample tests take their lists as one slice

`OneWayANOVA`、`KruskalWallis`、`LeveneTest`、`BartlettTest` SHALL 以一個 `[]insyra.IDataList` 接收各組；`RepeatedMeasuresANOVA` 與 `FriedmanTest` SHALL 以一個 `[]insyra.IDataList` 接收各受試者；`TwoWayANOVA` SHALL 以 `(factorALevels, factorBLevels int, cells []insyra.IDataList)` 接收各格。錯誤訊息中的組、格與受試者 SHALL 依 slice 中的順序從 1 編號。nil 或空的 slice SHALL 由原有的組數檢查回傳錯誤，SHALL NOT panic。

#### Scenario: Groups from a slice
- **WHEN** 呼叫 `OneWayANOVA([]insyra.IDataList{a, b, c})`
- **THEN** 結果與變更前 `OneWayANOVA(a, b, c)` 逐位元相同

#### Scenario: A nil slice
- **WHEN** 呼叫 `KruskalWallis(nil)`
- **THEN** 回傳錯誤，不 panic

### Requirement: Factor analysis runs without options

`FactorAnalysis` SHALL 以 `opts ...FactorAnalysisOptions` 接收設定。沒有給設定時 SHALL 使用 `DefaultFactorAnalysisOptions()` 的值；給了一個以上時 SHALL 回傳錯誤 `at most one stats.FactorAnalysisOptions may be given, got <個數>`。

#### Scenario: No options
- **WHEN** 呼叫 `FactorAnalysis(dt)`
- **THEN** 結果與 `FactorAnalysis(dt, DefaultFactorAnalysisOptions())` 相同

#### Scenario: Two options values
- **WHEN** 呼叫 `FactorAnalysis(dt, FactorAnalysisOptions{}, FactorAnalysisOptions{})`
- **THEN** 回傳錯誤，結果為 nil
