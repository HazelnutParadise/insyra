## ADDED Requirements

### Requirement: Every stats result prints itself the same way

`stats` 中每個名稱以 `Result` 結尾的匯出 struct 型別 SHALL 自己宣告 `String() string` 與 `Show()`（`FactorAnalysisResult` 為 `Show(startEndRange ...any)`），接收者為指標。內嵌 `FactorAnalysisResult` 的 `FactorModel` SHALL 自己宣告 `String()` 與 `Show(startEndRange ...any)`，交給內嵌的 `FactorAnalysisResult` 處理；透過內嵌取得的方法在 nil 的 `*FactorModel` 上會先解參考而 panic，所以不能只靠內嵌。`stats` 的測試 SHALL 解析套件原始碼，任何這類型別缺少自己的 `String` 或 `Show` 時失敗。

`String()` 的內容 SHALL 為：

- 第一行是分析的名稱（例如 `t-test`、`Two-way ANOVA`、`Factor analysis`）。
- 接著每個匯出欄位一行，格式為兩個空白、Go 欄位名稱、`: `、值，依宣告順序；內嵌的 `TestResult` 的欄位排在最前面。未匯出的欄位 SHALL NOT 出現。
- 值為 nil 的指標或介面欄位 SHALL 省略整行。非 nil 的指標顯示其指向的值。
- 浮點數 SHALL 依程式庫輸出文字的規則寫成文字：絕對值從 1e-6 到（不含）1e21 為一般小數，其外為指數形式，`NaN` 與 `±Inf` 為 `NaN`、`+Inf`、`-Inf`。
- 切片與陣列寫在同一行，形如 `[a b c]`；多於 60 個元素時 SHALL 只寫前 20 個與後 5 個，中間以 `...` 隔開，並在後面註明元素總數。巢狀的 struct 寫成 `{Field: value, …}`。
- 型別為表格（`insyra.IDataTable`）的欄位 SHALL 在欄位名稱那一行之下，以縮排的格線寫出列名、欄名與每一格；多於 60 列時 SHALL 只寫前 20 列與後 5 列，中間以 `...` 隔開。
- 輸出 SHALL NOT 含 ANSI 色碼，也 SHALL NOT 依終端機寬度改變，同一個結果在任何環境得到相同的文字。
- nil 接收者 SHALL 回傳 `<nil>`，SHALL NOT panic；`Show` 在 nil 接收者上 SHALL 印出 `<nil>`，有沒有給範圍都一樣。

`Show()` SHALL 把 `String()` 加一個換行寫到標準輸出。`FactorAnalysisResult.Show` 沒有給範圍時 SHALL 同樣印出 `String()`；給了範圍時 SHALL 照舊以該範圍顯示每張表。

#### Scenario: A t-test prints its fields
- **WHEN** 對 `SingleSampleTTest` 的結果呼叫 `String()`
- **THEN** 第一行為 `t-test`，接著依序有 `  Statistic: `、`  PValue: `、`  DF: `、`  CI: `、`  EffectSizes: `、`  Mean: `、`  N: ` 開頭的行，沒有 `Mean2`、`MeanDiff`、`N2`

#### Scenario: Large numbers are plain decimals
- **WHEN** 結果的某個欄位為 `1500000`
- **THEN** 該行寫成 `1500000`，不是 `1.5e+06`

#### Scenario: A long list is elided
- **WHEN** 對 100 個觀察值的 `LinearRegressionResult` 呼叫 `String()`
- **THEN** `Residuals` 那一行只列 25 個值並註明共 100 個

#### Scenario: A contingency table prints as a grid
- **WHEN** 對 `ChiSquareIndependenceTest` 的結果呼叫 `String()`
- **THEN** `Observed:` 與 `Expected:` 之下各有一個縮排的格線，含欄類別名稱、列類別名稱與每格的數字

#### Scenario: Show and String agree
- **WHEN** 把標準輸出導向緩衝區後呼叫 `res.Show()`
- **THEN** 緩衝區內容等於 `res.String() + "\n"`

#### Scenario: Any writer
- **WHEN** 呼叫 `fmt.Fprintln(&buf, res)`
- **THEN** `buf` 的內容等於 `res.String() + "\n"`
