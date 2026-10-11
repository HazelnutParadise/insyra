## ADDED Requirements

### Requirement: The two-way ANOVA and the repeated-measures tests take a long-format table

`stats` SHALL provide `TwoWayANOVAFromTable`, `RepeatedMeasuresANOVAFromTable`, and `FriedmanTestFromTable` accepting long-format tables with string column selectors.

#### Scenario: A two-way table
- **WHEN** 表格有 `score`、`drug`、`dose` 三欄，每列一個觀察值，呼叫 `TwoWayANOVAFromTable(dt, "score", "drug", "dose")`
- **THEN** 結果與把各組合的分數依 `drug`、`dose` 第一次出現的順序排成 row-major 的 `cells` 後呼叫 `TwoWayANOVA` 完全相同

#### Scenario: Column selection by letter first then name
- **WHEN** 表格欄位同時存在符合 Excel 字母之欄位與欄名
- **THEN** 優先解析為既有欄位之 Excel 字母，若無則依完全相符之欄名解析

#### Scenario: Integer levels of different widths are one level
- **WHEN** 條件欄一半的值是 `int64(1)`、另一半是 `int(1)`，其餘為 `int64(2)`
- **THEN** 依 `ToMapKey` 比較只有兩個條件水準

#### Scenario: The caller's table keeps no error
- **WHEN** 以超出範圍或不存在的 `valueCol` 呼叫任一函式
- **THEN** 回傳錯誤，呼叫端表格的 `Err()` 仍為 nil

### Requirement: Long-format input is refused, not repaired

Long-format entry points SHALL validate numeric inputs, levels, and combinations before allocating cells and return descriptive errors on invalid data.

#### Scenario: A missing measurement
- **WHEN** 受試者 `s3` 在條件 `t2` 沒有任何列
- **THEN** 回傳錯誤 `subject s3 has no observation for condition t2`

#### Scenario: A blank level
- **WHEN** 第 5 列的受試者欄是 nil
- **THEN** 回傳錯誤 `subject column <選擇器> has no level at row 5`

#### Scenario: Check level combinations before allocating
- **WHEN** 輸入資料之水準組合不完全或存在重複觀察值
- **THEN** 函式在配置網格記憶體前即拒絕並回傳錯誤

### Requirement: The long-format entry points agree with R

Long-format entry points SHALL produce results matching R reference calculations on shuffled row order data.

#### Scenario: Shuffled rows
- **WHEN** 以 R 腳本打亂列順序後輸出的長格式資料建立表格並呼叫三個函式
- **THEN** 平方和、自由度、F 值、統計量與 p 值都與 R 的輸出相符
