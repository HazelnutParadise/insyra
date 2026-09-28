## ADDED Requirements

### Requirement: fillna counts only the chosen kind of missing toward limit

With `missing nan` or `missing nil`, `fillna`'s `ffill` and `bfill` SHALL skip the cells of the other kind: they SHALL NOT be filled, SHALL NOT provide a value to copy, and SHALL NOT count toward `limit`. Without a `limit`, and for `mean`, `median`, `mode` and `interpolate`, the result SHALL be what it was.

#### Scenario: A nil before the NaN does not use up the limit
- **WHEN** 變數 `x` 為 `[1, nil, NaN]`，執行 `fillna x ffill limit 1 missing nan as f`
- **THEN** `f` 為 `[1, nil, 1]`

#### Scenario: The same backwards for nil
- **WHEN** 變數 `x` 為 `[nil, NaN, 5]`，執行 `fillna x bfill limit 1 missing nil as f`
- **THEN** `f` 的第 1 格為 5、第 2 格仍為 NaN、第 3 格為 5

#### Scenario: Only target cells count toward the limit
- **WHEN** 變數 `x` 為 `[1, NaN, nil, NaN]`，執行 `fillna x ffill limit 1 missing nan as f`
- **THEN** `f` 為 `[1, 1, nil, NaN]`

#### Scenario: A table column follows the same rule
- **WHEN** 表格欄 `v` 為 `[1, nil, NaN]`，執行 `fillna t ffill cols name:v limit 1 missing nan as f`
- **THEN** `f` 的 `v` 欄為 `[1, nil, 1]`
