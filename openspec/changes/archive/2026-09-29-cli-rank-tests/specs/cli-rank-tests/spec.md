## ADDED Requirements

### Requirement: The CLI runs the Wilcoxon, Mann-Whitney U and Kruskal-Wallis tests

CLI SHALL 提供：

- `wilcoxon single <var> <mu> [two-sided|greater|less]`，執行 `stats.SingleSampleWilcoxon`；`wilcoxon paired <var1> <var2> [two-sided|greater|less]`，執行 `stats.PairedWilcoxon`；兩者印出 `W=<統計量> p=<p 值>`。
- `mannwhitney <var1> <var2> [two-sided|greater|less]`，執行 `stats.MannWhitneyU`，印出 `U=<統計量> p=<p 值>`。
- `kruskal <group1> <group2> [groupN]`，執行 `stats.KruskalWallis`，印出 `H=<統計量> df=<自由度> p=<p 值>`。

對立假設 SHALL 以 `ztest` 使用的同一個解析器讀取；省略時為雙尾；無法辨識的寫法 SHALL 回傳錯誤，SHALL NOT 當成雙尾執行。程式庫回傳的錯誤 SHALL 以 `<指令> failed: <錯誤>` 回報。參數數量 SHALL 依各指令宣告檢查。`Usage`、`Forms`、`Examples` 與 `Docs/cli-dsl.md` 的指令索引與指令分組 SHALL 列出三個指令。

#### Scenario: Output matches the library
- **WHEN** 對同一份資料執行 `wilcoxon paired before after less`
- **THEN** 輸出的 `W` 與 `p` 等於 `stats.PairedWilcoxon(before, after, stats.WilcoxonOptions{Alternative: stats.Less})` 的 `Statistic` 與 `PValue`

#### Scenario: A misspelled alternative is refused
- **WHEN** 執行 `mannwhitney a b grater`
- **THEN** 指令回傳錯誤，不輸出結果
