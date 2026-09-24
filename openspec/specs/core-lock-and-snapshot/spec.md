# core-lock-and-snapshot Specification

## Purpose
A call that locks or snapshots data protects what it says it protects: AtomicDoAll locks every instance it can and skips a nil one without panicking, and GroupBy aggregates the table it grouped rather than whatever the parent holds later.

## Requirements

### Requirement: AtomicDoAll skips a nil instance

`AtomicDoAll` SHALL skip a nil `*DataList`, a nil `*DataTable` and a nil value without panicking, SHALL still run the callback, and SHALL still lock every other instance it was given.

#### Scenario: A nil instance
- **WHEN** 呼叫 `AtomicDoAll(f, dl, nilList)`
- **THEN** `f` 執行且不 panic，`dl` 仍被鎖住

### Requirement: GroupBy aggregates the table it grouped

`GroupBy` SHALL take a copy of the column data it groups. `Aggregate` SHALL compute from that copy, and SHALL NOT read the parent table's data.

#### Scenario: The parent changes between GroupBy and Aggregate
- **WHEN** GroupBy 之後修改父表的值，再呼叫 Aggregate
- **THEN** 結果反映 GroupBy 當下的資料，且在 `-race` 下沒有 data race
