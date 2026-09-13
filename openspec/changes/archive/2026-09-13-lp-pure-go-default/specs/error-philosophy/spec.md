## ADDED Requirements

### Requirement: A failing function still returns a usable DataTable

回傳 `*insyra.DataTable` 的函式 SHALL NOT 在失敗時回傳 nil，因為呼叫端的下一行通常就會呼叫它的方法而 panic。失敗時 SHALL 回傳空但可用的表格，並把原因記在 `Err()` 上。

#### Scenario: A failure returns an empty table
- **WHEN** 回傳 `*insyra.DataTable` 的函式失敗
- **THEN** 回傳的表格不為 nil，呼叫 `Show()` 不會 panic，`Err()` 說明失敗原因

## REMOVED Requirements

### Requirement: A function returning a DataTable returns a usable one

**Reason**: Both of its scenarios tested `lp.SolveFromFile`, which no longer exists. The rule itself continues word for word as "A failing function still returns a usable DataTable", with a scenario that names no removed function.

**Migration**: Nothing for callers. `lp.Solve` and `lp.SolveFile` return `(*Solution, error)`, and `Solution.ToDataTable` follows the continuing rule.

### Requirement: A reported success means there is a result

**Reason**: The additional-info table and its `Status` string are gone. `lp.Solve` and `lp.SolveFile` return a `Solution` or an error, never a success string beside an unreadable result; `lp-solve` states that rule.

**Migration**: Check the returned error, then `Solution.Status`.
