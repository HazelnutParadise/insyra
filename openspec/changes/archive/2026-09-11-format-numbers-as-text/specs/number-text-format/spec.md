## ADDED Requirements

### Requirement: A number is written as text by one rule

When insyra writes a `float64` or `float32` as text for output — CCL string functions, `ToCSV`, `DataList.ToStringSlice`, and column names generated from values — it SHALL write a plain decimal for magnitudes from 1e-6 up to but not including 1e21, and SHALL use exponent form outside that range, with the same text Go's shortest exponent formatting produces today. `NaN`, `+Inf` and `-Inf` SHALL be written as `NaN`, `+Inf` and `-Inf`.

#### Scenario: A revenue in the millions
- **WHEN** CCL 求值 `'營收：' & A * B`，其中 A 為 1500、B 為 1000
- **THEN** 得到 `"營收：1500000"`，而不是 `"營收：1.5e+06"`

#### Scenario: The same column exported
- **WHEN** 對同一個計算欄執行 `ToCSV`
- **THEN** 檔案寫的是 `1500000`，讀回來與原本的 `float64` 完全相等

#### Scenario: A value that stays in exponent form
- **WHEN** 寫出 `0.0000001` 或 `1e21`
- **THEN** 分別得到 `1e-07` 與 `1e+21`，與改動前逐字相同

### Requirement: Text that is not output keeps its current form

The string comparison used when sorting mixed types, cache keys, and the terminal display of `Show` SHALL NOT change as part of this rule. The name given to a `nil` category by `OneHotEncode` and `Pivot` SHALL remain `<nil>`.

#### Scenario: Sorting mixed types
- **WHEN** 對混合型別的欄位排序
- **THEN** 排序結果與改動前相同
