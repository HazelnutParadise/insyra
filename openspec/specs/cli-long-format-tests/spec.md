# cli-long-format-tests Specification

## Purpose
The CLI forms that run the two-way ANOVA, the repeated-measures ANOVA and the Friedman test on a loaded table with one row per observation, picked by the first argument's variable type so the list forms keep their meaning, and the `friedman` command itself.

## Requirements

### Requirement: anova twoway and anova repeated take a table with one row per observation

`anova twoway` SHALL take the table form, `anova twoway <table> <value> <factorA> <factorB>`, when its first argument after the mode is a DataTable variable, unless its first two arguments are both integers, which select the list form. `anova repeated` SHALL take the table form when its first argument is a DataTable variable. Otherwise both SHALL run the list form as before. The table form SHALL print the same output as the list form.

#### Scenario: Two-way ANOVA from a table
- **WHEN** a table `t` has columns `score`, `drug` and `dose`, and `anova twoway t score drug dose` is run
- **THEN** the output matches the `FactorA` and `FactorB` results of `stats.TwoWayANOVAFromTable(t, Name("score"), Name("drug"), Name("dose"))`, and of `anova twoway 2 2 …` run on the same data split into lists in first-appearance order

#### Scenario: The list form is unchanged
- **WHEN** `anova twoway 2 2 c11 c12 c21 c22` is run and the four variables are DataLists
- **THEN** the output is the same as before this change

#### Scenario: A table named like a number
- **WHEN** a DataTable is stored under the name `2` and `anova twoway 2 2 c11 c12 c21 c22` is run with four DataList variables
- **THEN** the list form runs with two levels for each factor, and the table is not used

### Requirement: The table form of anova reads its columns by the CLI token rule

欄位參數 SHALL 依 CLI 的單一 token 規則解析（#315）：純 token 同時讀成從 0 起算的編號、字母索引與欄名，讀法指向不同欄時拒絕，`number:`、`index:`、`name:` 指定唯一讀法。表格形式 SHALL 恰好收四個參數；多或少時 SHALL 回傳該形式的用法錯誤。程式庫回傳的錯誤 SHALL 以 `anova failed: <錯誤>` 回報。

#### Scenario: A missing measurement
- **WHEN** 表格中受試者 `s3` 沒有條件 `t2` 的列，執行 `anova repeated t value cond subj`
- **THEN** 指令回傳錯誤，訊息含 `subject s3 has no observation for condition t2`

### Requirement: The CLI has a Friedman test

CLI SHALL 提供 `friedman` 指令，有兩種形式：第一個參數是 DataTable 變數時為 `friedman <table> <value> <condition> <subject>`，執行 `stats.FriedmanTestFromTable`；否則為 `friedman <subject1> <subject2> [subjectN]`，每個參數是一位受試者的 DataList，執行 `stats.FriedmanTest`。欄位參數 SHALL 依單一 token 規則解析，表格形式 SHALL 恰好收四個參數。指令 SHALL 印出 `Q=<統計量> df=<自由度> p=<p 值>`；程式庫的錯誤 SHALL 以 `friedman failed: <錯誤>` 回報。`Usage`、`Forms`、`Examples` 與 `Docs/cli-dsl.md` 的指令索引與指令分組 SHALL 列出它。

#### Scenario: Friedman from a table and from lists
- **WHEN** 同一份資料分別以 `friedman t value cond subj` 與每位受試者一個 DataList 的 `friedman s1 s2 s3 …` 執行
- **THEN** 兩者的輸出相同，也與 `stats.FriedmanTestFromTable` 的 `Statistic`、`DF`、`PValue` 相同
