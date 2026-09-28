# stats-long-format Specification

## Purpose
The long-format entry points to the two-way ANOVA, the repeated-measures ANOVA and the Friedman test: a table with one row per observation, its columns picked by the one column selector rule, grouped and handed to the list-taking functions so the numbers come from one implementation, and refused rather than repaired when a level is missing or a design cell is empty.
## Requirements
### Requirement: The two-way ANOVA and the repeated-measures tests take a long-format table

`stats` SHALL 提供：

- `TwoWayANOVAFromTable(dt insyra.IDataTable, valueCol, factorACol, factorBCol any) (*TwoWayANOVAResult, error)`
- `RepeatedMeasuresANOVAFromTable(dt insyra.IDataTable, valueCol, conditionCol, subjectCol any) (*RepeatedMeasuresANOVAResult, error)`
- `FriedmanTestFromTable(dt insyra.IDataTable, valueCol, conditionCol, subjectCol any) (*FriedmanTestResult, error)`

表格的每一列是一個觀察值。每個欄位參數 SHALL 依程式庫唯一的欄位選擇規則解讀：字串為 Excel 式索引，`insyra.Name(...)` 為欄名，`int` 為從 0 起算的位置。選擇器無法解析時 SHALL 回傳說明哪個參數、原因為何的錯誤。這些函式 SHALL NOT 修改呼叫端的表格，也 SHALL NOT 在其 `Err()` 留下錯誤。`dt` 為 nil 時 SHALL 回傳錯誤，SHALL NOT panic。

因子、條件與受試者欄的相異值是該欄的水準，比較方式與 `insyra.ToMapKey` 相同：各種寬度的整數依數值相等，浮點數與文字各自獨立。水準依第一次出現的順序排列。

結果 SHALL 與以同樣分組、依水準第一次出現的順序呼叫 `TwoWayANOVA`、`RepeatedMeasuresANOVA` 或 `FriedmanTest` 的結果逐位元相同；那三個函式的行為 SHALL 不變。

#### Scenario: A two-way table
- **WHEN** 表格有 `score`、`drug`、`dose` 三欄，每列一個觀察值，呼叫 `TwoWayANOVAFromTable(dt, insyra.Name("score"), insyra.Name("drug"), insyra.Name("dose"))`
- **THEN** 結果與把各組合的分數依 `drug`、`dose` 第一次出現的順序排成 row-major 的 `cells` 後呼叫 `TwoWayANOVA` 完全相同

#### Scenario: Integer levels of different widths are one level
- **WHEN** 條件欄一半的值是 `int64(1)`、另一半是 `int(1)`，其餘為 `int64(2)`
- **THEN** 只有兩個條件水準

#### Scenario: The caller's table keeps no error
- **WHEN** 以超出範圍的 `valueCol` 呼叫任一函式
- **THEN** 回傳錯誤，呼叫端表格的 `Err()` 仍為 nil

### Requirement: Long-format input is refused, not repaired

- 數值欄的每一格 SHALL 是有限數值；否則 SHALL 回傳 `value column <選擇器> contains a non-numeric value at row <列>: <值>` 或 `value column <選擇器> contains a non-finite value at row <列>: <值>`，列從 1 起算。
- 因子、條件或受試者欄的格為 nil 或 `NaN` 時 SHALL 回傳 `<角色> column <選擇器> has no level at row <列>`，列從 1 起算，角色為 `factor A`、`factor B`、`condition` 或 `subject`。
- `TwoWayANOVAFromTable`：任一因子少於兩個水準時 SHALL 回傳 `factor A has fewer than two levels`（或 `factor B`）；任一水準組合沒有觀察值時 SHALL 回傳 `no observations for A=<水準>, B=<水準>`。
- `RepeatedMeasuresANOVAFromTable` 與 `FriedmanTestFromTable`：同一受試者在同一條件有兩個以上觀察值時 SHALL 回傳 `subject <受試者> has more than one observation for condition <條件> (rows <列> and <列>)`；受試者缺少某條件的觀察值時 SHALL 回傳 `subject <受試者> has no observation for condition <條件>`。條件或受試者少於兩個時 SHALL 回傳錯誤。

這些情況都 SHALL NOT 回傳結果，也 SHALL NOT 略過該列後繼續計算。

#### Scenario: A missing measurement
- **WHEN** 受試者 `s3` 在條件 `t2` 沒有任何列
- **THEN** 回傳錯誤 `subject s3 has no observation for condition t2`

#### Scenario: A blank level
- **WHEN** 第 5 列的受試者欄是 nil
- **THEN** 回傳錯誤 `subject column <選擇器> has no level at row 5`

### Requirement: The long-format entry points agree with R

`TwoWayANOVAFromTable` SHALL 在平衡設計上與 R `aov(value ~ A * B)` 相符；`RepeatedMeasuresANOVAFromTable` SHALL 與 R `aov(value ~ cond + Error(subj/cond))` 相符；`FriedmanTestFromTable` SHALL 與 R `friedman.test(value ~ cond | subj)` 相符。比對使用列順序打亂、水準為文字的長格式資料，參考值由呼叫這些 R 函式的腳本產生並提交到 `stats/testdata`，測試執行時不需要 R。

#### Scenario: Shuffled rows
- **WHEN** 以 R 腳本打亂列順序後輸出的長格式資料建立表格並呼叫三個函式
- **THEN** 平方和、自由度、F 值、統計量與 p 值都與 R 的輸出相符

