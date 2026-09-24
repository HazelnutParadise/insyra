## ADDED Requirements

### Requirement: A reference baseline that draws random numbers is seeded

產生參考值的腳本若會呼叫使用亂數的參考實作（例如 `psych::fa` 的多個起點），SHALL 在呼叫前以固定的種子設定亂數產生器，使同一組輸入在任何工作階段都產生完全相同的輸出。快取的參考值 SHALL NOT 取決於建立快取時的工作階段亂數狀態。

#### Scenario: The same payload twice
- **WHEN** 以同一組因素分析輸入（`two_blocks`、ML 抽取、oblimin、迴歸分數、兩個因子）不經快取執行參考值腳本兩次
- **THEN** 兩次的輸出逐位元組相同
