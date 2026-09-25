# lint-coverage Specification

## Purpose
CI 在 golangci-lint 預設組合之外多跑哪些 linter、為什麼：專門盯「借了資源沒還」與「錯誤被吃掉或講不清楚」這兩類問題，並保證 `env import` 的覆蓋前檢查在讀不到檔案時拒絕而不是放行。`gosec`、`unparam`、`noctx` 經實測雜訊遠多於訊號，刻意不開。
## Requirements
### Requirement: CI lints for resources left open and errors lost

The lint configuration SHALL enable `nilerr`, `bodyclose`, `rowserrcheck`, `sqlclosecheck` and `errorlint` in addition to golangci-lint's default set, and the repository SHALL have no finding from them. A finding judged a false positive SHALL be suppressed where it is reported, with the reason beside it.

#### Scenario: An error formatted without wrapping
- **WHEN** code passes an error to `fmt.Errorf` with `%v`
- **THEN** the lint job fails

#### Scenario: A guard that returns nil for a failure
- **WHEN** a function returns a nil error on a branch where it has a non-nil one
- **THEN** the lint job fails

### Requirement: An environment that cannot be read is not overwritten without force

`insyra env import` without `--force` SHALL refuse a target environment whose `state.json`, `history.txt` or `config.json` exists but cannot be read. A missing file SHALL count as empty.

#### Scenario: config.json cannot be read
- **WHEN** the target's `config.json` exists but reading it fails
- **THEN** import without `--force` fails and the environment is unchanged

#### Scenario: An environment with nothing in it
- **WHEN** the target was just created and holds no files
- **THEN** import without `--force` succeeds

