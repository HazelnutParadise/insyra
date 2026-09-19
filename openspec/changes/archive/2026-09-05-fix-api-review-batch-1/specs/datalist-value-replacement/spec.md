## ADDED Requirements

### Requirement: Replace methods touch only cells equal to oldValue

`DataList.ReplaceFirst`、`ReplaceLast`、`ReplaceAll` SHALL 只改寫與 `oldValue` 相等的格子。`oldValue` 是 NaN 時 SHALL 以「格子是 float64 NaN」作為相等條件；`oldValue` 不是 NaN 時，NaN 格子 SHALL NOT 被視為相等。`ReplaceLast` SHALL 從尾端往前找第一個相等的格子並只改寫該格。

#### Scenario: ReplaceLast leaves a trailing NaN alone

- **WHEN** `NewDataList(5, math.NaN()).ReplaceLast(5, 0)`
- **THEN** 資料為 `[0, NaN]`

#### Scenario: ReplaceLast replaces the last match, not the first

- **WHEN** `NewDataList(5, 1, 5).ReplaceLast(5, 0)`
- **THEN** 資料為 `[5, 1, 0]`

#### Scenario: ReplaceLast with NaN oldValue targets the last NaN

- **WHEN** `NewDataList(math.NaN(), 1, math.NaN()).ReplaceLast(math.NaN(), 0)`
- **THEN** 資料為 `[NaN, 1, 0]`
