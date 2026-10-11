## ADDED Requirements

### Requirement: `@` in ApplyCCL holds each row's own values

`ApplyCCL` SHALL give each row's `@` the values of that row, so that a column built from `@` differs from row to row as the file does.

#### Scenario: Each row's current values
- **WHEN** 對 `parquet/ccl_test.go` 的四列測試檔執行 `ApplyCCL(ctx, path, "NEW('r') = @")` 後讀回
- **THEN** `r` 依序為 `[1 10 a true]`、`[2 20 b false]`、`[3 30 c true]`、`[4 40 d false]`，每列都是自己那一列的值
