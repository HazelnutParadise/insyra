## ADDED Requirements

### Requirement: A rounding mode does not panic

`finance` 的 `RoundUnnecessary` 在結果需要捨入時 SHALL 回傳錯誤，SHALL NOT panic。該模式的用途是得知捨入發生了，回報比中止程序更能達成這件事。

#### Scenario: A result that has to be rounded
- **WHEN** `NPV(0.03, []{0, 1}, Options{Scale: 2, Mode: RoundUnnecessary})`
- **THEN** 回傳說明需要捨入的錯誤，程序繼續執行
