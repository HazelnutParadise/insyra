## ADDED Requirements

### Requirement: anova twoway and anova repeated take a table with one row per observation

`anova twoway` SHALL 在模式後的第一個參數是 DataTable 變數時，以 `anova twoway <table> <value> <factorA> <factorB>` 執行 `stats.TwoWayANOVAFromTable`；`anova repeated` SHALL 在模式後的第一個參數是 DataTable 變數時，以 `anova repeated <table> <value> <condition> <subject>` 執行 `stats.RepeatedMeasuresANOVAFromTable`。第一個參數不是 DataTable 變數時，兩者 SHALL 照舊執行清單形式，行為與輸出不變。

欄位參數 SHALL 依 CLI 的單一 token 規則解析（#315）：純 token 同時讀成從 0 起算的編號、字母索引與欄名，讀法指向不同欄時拒絕，`number:`、`index:`、`name:` 指定唯一讀法。表格形式 SHALL 恰好收四個參數；多或少時 SHALL 回傳該形式的用法錯誤。程式庫回傳的錯誤 SHALL 以 `anova failed: <錯誤>` 回報。輸出 SHALL 與清單形式相同：`anova twoway` 印 `FA=… pA=… FB=… pB=…`，`anova repeated` 印 `F=… p=…`。

#### Scenario: Two-way ANOVA from a table
- **WHEN** 表格 `t` 有 `score`、`drug`、`dose` 三欄，執行 `anova twoway t score drug dose`
- **THEN** 輸出與 `stats.TwoWayANOVAFromTable(t, Name("score"), Name("drug"), Name("dose"))` 的 `FactorA`、`FactorB` 相同，也與把同樣資料依水準第一次出現的順序切成清單後執行的 `anova twoway 2 2 …` 相同

#### Scenario: The list form is unchanged
- **WHEN** 執行 `anova twoway 2 2 c11 c12 c21 c22`，四個變數都是 DataList
- **THEN** 輸出與變更前相同

#### Scenario: A missing measurement
- **WHEN** 表格中受試者 `s3` 沒有條件 `t2` 的列，執行 `anova repeated t value cond subj`
- **THEN** 指令回傳錯誤，訊息含 `subject s3 has no observation for condition t2`

### Requirement: The CLI has a Friedman test

CLI SHALL 提供 `friedman` 指令，有兩種形式：第一個參數是 DataTable 變數時為 `friedman <table> <value> <condition> <subject>`，執行 `stats.FriedmanTestFromTable`；否則為 `friedman <subject1> <subject2> [subjectN]`，每個參數是一位受試者的 DataList，執行 `stats.FriedmanTest`。欄位參數 SHALL 依單一 token 規則解析，表格形式 SHALL 恰好收四個參數。指令 SHALL 印出 `Q=<統計量> df=<自由度> p=<p 值>`；程式庫的錯誤 SHALL 以 `friedman failed: <錯誤>` 回報。`Usage`、`Forms`、`Examples` 與 `Docs/cli-dsl.md` 的指令索引與指令分組 SHALL 列出它。

#### Scenario: Friedman from a table and from lists
- **WHEN** 同一份資料分別以 `friedman t value cond subj` 與每位受試者一個 DataList 的 `friedman s1 s2 s3 …` 執行
- **THEN** 兩者的輸出相同，也與 `stats.FriedmanTestFromTable` 的 `Statistic`、`DF`、`PValue` 相同
