# cli-long-format-tests Specification

## Purpose
How the CLI's `anova twoway`, `anova repeated` and `friedman` commands run on a table with one row per observation, and how that form is told apart from the list forms.

## Requirements

### Requirement: anova twoway and anova repeated take a table with one row per observation

`anova twoway` and `anova repeated` SHALL support long-format table inputs while preserving list forms when arguments are integer level counts or DataLists.

#### Scenario: Two-way ANOVA from a table
- **WHEN** 表格 `t` 有 `score`、`drug`、`dose` 三欄，執行 `anova twoway t score drug dose`
- **THEN** 輸出與 `stats.TwoWayANOVAFromTable(t, "score", "drug", "dose")` 的 `FactorA`、`FactorB` 相同，也與切成清單後執行的 `anova twoway 2 2 …` 相同

#### Scenario: Integer first argument prefers list form
- **WHEN** 存在名為 "2" 的 DataTable 變數，但執行 `anova twoway 2 2 c11 c12 c21 c22`
- **THEN** 第一個參數解析為整數水準數，執行清單形式而非表格形式

#### Scenario: The list form is unchanged
- **WHEN** 執行 `anova twoway 2 2 c11 c12 c21 c22`，四個變數都是 DataList
- **THEN** 輸出與變更前相同

#### Scenario: A missing measurement
- **WHEN** 表格中受試者 `s3` 沒有條件 `t2` 的列，執行 `anova repeated t value cond subj`
- **THEN** 指令回傳錯誤，訊息含 `subject s3 has no observation for condition t2`

### Requirement: The CLI has a Friedman test

CLI SHALL provide a `friedman` command supporting both long-format DataTable input and per-subject DataList input.

#### Scenario: Friedman from a table and from lists
- **WHEN** 同一份資料分別以 `friedman t value cond subj` 與每位受試者一個 DataList 的 `friedman s1 s2 s3 …` 執行
- **THEN** 兩者的輸出相同，也與 `stats.FriedmanTestFromTable` 的 `Statistic`、`DF`、`PValue` 相同
