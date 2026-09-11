## ADDED Requirements

### Requirement: Commands reject arguments they do not use

The DataList statistics commands (`sum`, `mean`, `median`, `mode`, `stdev`, `var`, `min`, `max`, `range`) and `accel` SHALL return an error naming the command and the argument when given an argument they do not use, and SHALL NOT perform the action.

#### Scenario: An alias on a command that stores nothing
- **WHEN** 執行 `mean x as m`
- **THEN** 回傳指出 `mean` 與 `"as"` 的錯誤，而且不會建立變數 `m`

#### Scenario: A removed accel flag
- **WHEN** 執行 `accel plan --precision float32`
- **THEN** 回傳錯誤，不產生規劃報告

### Requirement: accel advertises only what it reads

Besides its action, `accel`'s Usage and registered flags SHALL list only `--mode`.

#### Scenario: help accel
- **WHEN** 執行 `help accel`
- **THEN** Usage 不含 `--precision`，Cobra 也沒有註冊 `--precision`
