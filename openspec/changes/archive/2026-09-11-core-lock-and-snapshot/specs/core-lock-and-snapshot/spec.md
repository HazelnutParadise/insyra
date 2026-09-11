## ADDED Requirements

### Requirement: AtomicDoAll only accepts values it can lock

`AtomicDoAll` SHALL accept only values that implement `Lockable`: `*DataList`, `*DataTable`, and types that embed them. A value it cannot lock SHALL be a compile error rather than a warning followed by an unlocked callback. A nil instance SHALL be skipped without panicking.

#### Scenario: A nil instance
- **WHEN** 呼叫 `AtomicDoAll(f, dl, nilList)`
- **THEN** `f` 執行且不 panic，`dl` 仍被鎖住

#### Scenario: A wrapper type
- **WHEN** 把 `isr` 的包裝型別傳給 `AtomicDoAll`
- **THEN** 能通過編譯並被鎖住

### Requirement: GroupBy aggregates the table it grouped

`GroupBy` SHALL take a copy of the column data it groups. `Aggregate` SHALL compute from that copy, and SHALL NOT read the parent table's data.

#### Scenario: The parent changes between GroupBy and Aggregate
- **WHEN** GroupBy 之後修改父表的值，再呼叫 Aggregate
- **THEN** 結果反映 GroupBy 當下的資料，且在 `-race` 下沒有 data race

### Requirement: ExecuteCCL applies all statements or none

`ExecuteCCL` SHALL leave the table unchanged when any statement fails. When every statement succeeds, the result SHALL equal applying the statements in order, each seeing the results of those before it.

#### Scenario: The third of four statements fails
- **WHEN** 腳本的第三條語句在執行期失敗
- **THEN** 表的欄位與資料與呼叫前完全相同，`Err()` 指出失敗的語句

### Requirement: Tests do not leak configuration

A test that changes the global `Config` SHALL restore the previous values when it finishes. A package's baseline configuration SHALL be set in one visible place.

#### Scenario: A test quietens logging
- **WHEN** 測試把 log level 設為 Fatal
- **THEN** 測試結束時恢復成原本的值
