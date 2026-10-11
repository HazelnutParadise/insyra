## ADDED Requirements

### Requirement: The CLI runs the Wilcoxon, Mann-Whitney U and Kruskal-Wallis tests

CLI SHALL provide `wilcoxon`, `mannwhitney`, and `kruskal` commands matching library results, supporting negative mu in one-shot mode, and rejecting unread trailing arguments.

#### Scenario: Output matches the library
- **WHEN** 對同一份資料執行 `wilcoxon paired before after less`
- **THEN** 輸出的 `W` 與 `p` 等於 `stats.PairedWilcoxon(before, after, stats.WilcoxonOptions{Alternative: stats.Less})` 的 `Statistic` 與 `PValue`

#### Scenario: A misspelled alternative is refused
- **WHEN** 執行 `mannwhitney a b grater`
- **THEN** 指令回傳錯誤，不輸出結果

#### Scenario: Negative mu in one-shot mode
- **WHEN** 在 one-shot 模式執行 `wilcoxon single x -1` 且 mu 為負數
- **THEN** 負數參數被正確解析為 mu 數值而非旗標

#### Scenario: Extra arguments are refused
- **WHEN** 執行 `mannwhitney a b two-sided greater` 帶有多餘參數
- **THEN** 指令回傳 usage 錯誤，拒絕執行
