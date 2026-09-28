## MODIFIED Requirements

### Requirement: Hypothesis tests take their alternative and confidence level in an optional options struct

`SingleSampleTTest`、`TwoSampleTTest`、`PairedTTest` SHALL 以最後一個參數 `opts ...TTestOptions` 接收設定；`SingleSampleZTest`、`TwoSampleZTest` SHALL 以 `opts ...ZTestOptions` 接收；`SingleSampleWilcoxon`、`PairedWilcoxon` SHALL 以 `opts ...WilcoxonOptions` 接收；`MannWhitneyU` SHALL 以 `opts ...MannWhitneyUOptions` 接收。這四個型別 SHALL 各有 `Alternative AlternativeHypothesis` 與 `ConfidenceLevel float64` 兩個欄位；`TTestOptions` SHALL 另有 `EqualVariance bool`，只有 `TwoSampleTTest` 讀取它，`SingleSampleTTest` 與 `PairedTTest` SHALL 忽略它。

`Alternative` 為空字串時 SHALL 視為 `TwoSided`；為 `TwoSided`、`Greater` 或 `Less` 以外的值時 SHALL 回傳錯誤 `alternative must be two-sided, greater or less, got "<值>"`。`ConfidenceLevel` 為 0 時 SHALL 視為 0.95；嚴格介於 0 與 1 之間時 SHALL 照用；其他值，包括 `NaN`、負數與大於等於 1 的值，SHALL 回傳錯誤 `confidence level must be strictly between 0 and 1, got <值>`。給了一個以上的設定值時 SHALL 回傳錯誤 `at most one stats.<型別> may be given, got <個數>`。這些檢查 SHALL 在讀取任何樣本之前完成，出錯時 SHALL NOT 回傳結果。

`mu`、`sigma`、`sigma1`、`sigma2` SHALL 維持為必填的位置參數。`TwoSampleTTest` SHALL 為 `TwoSampleTTest(data1, data2 insyra.IDataList, opts ...TTestOptions)`，SHALL NOT 另有表示變異數假設的位置參數。

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

## ADDED Requirements

### Requirement: The two-sample t-test runs Welch's test unless told otherwise

`TwoSampleTTest` 在沒有給設定值或 `EqualVariance` 為 false 時 SHALL 執行 Welch t 檢定（不合併變異數，自由度為 Welch-Satterthwaite），與 R `t.test(x, y)` 的預設相同；`EqualVariance` 為 true 時 SHALL 執行合併變異數的 Student t 檢定，自由度為 `n1 + n2 − 2`，與 R `t.test(x, y, var.equal = TRUE)` 相同。CLI 的 `ttest two <var1> <var2>` 沒有給變異數寫法時 SHALL 執行 Welch t 檢定；`equal` 與 `unequal` 的意義 SHALL 不變。

#### Scenario: Default is Welch
- **WHEN** 呼叫 `TwoSampleTTest(x, y)`，`x` 為 `55.1, 49.3, 58.2, 61.9, 47.3, 51.0, 53.8, 59.7`，`y` 為 `46.9, 41.2, 45.7, 49.8, 44.0, 47.6, 46.5, 43.9, 50.2`
- **THEN** 統計量為 4.03179446342583，自由度為 10.70275033671886，p 值為 0.00208702339328（R `t.test(x, y)`），與 `TwoSampleTTest(x, y, TTestOptions{EqualVariance: false})` 逐位元相同

#### Scenario: Equal variances on request
- **WHEN** 呼叫 `TwoSampleTTest(x, y, TTestOptions{EqualVariance: true})`
- **THEN** 統計量為 4.16693760749，自由度為 15，p 值為 0.000826315295455（R `t.test(x, y, var.equal = TRUE)`）

#### Scenario: CLI default
- **WHEN** 執行 `ttest two a b`
- **THEN** 輸出與 `ttest two a b unequal` 相同，與 `ttest two a b equal` 不同
