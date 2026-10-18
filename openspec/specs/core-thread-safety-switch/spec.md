# core-thread-safety-switch Specification

## Purpose
What `Config.Dangerously_TurnOffThreadSafety` turns off, the per-instance lock of DataList and DataTable, and how `Config.TurnOnThreadSafety` restores it without touching the other settings.

## Requirements

### Requirement: Thread safety can be turned back on alone

`Config.TurnOnThreadSafety()` SHALL make `DataList.AtomicDo`, `DataTable.AtomicDo` and `AtomicDoAll` lock again after `Dangerously_TurnOffThreadSafety`, and SHALL leave every other `Config` setting as it was.

#### Scenario: Other settings survive
- **WHEN** 設定 log 等級為 Error、關閉彩色輸出與加速，呼叫 `Dangerously_TurnOffThreadSafety()` 後再呼叫 `TurnOnThreadSafety()`
- **THEN** log 等級仍是 Error、彩色輸出與加速仍是關閉

#### Scenario: Locking works again
- **WHEN** 在關閉 thread safety 時建立一個 DataList，再呼叫 `TurnOnThreadSafety()`，然後 64 個 goroutine 各 `Append` 100 次
- **THEN** 列表長度是 6400，race detector 不回報

### Requirement: Turning thread safety back on is logged once

`TurnOnThreadSafety` SHALL log at Info level when locking was off, and SHALL log nothing when locking is already on.

#### Scenario: A transition and a repeat
- **WHEN** 關閉 thread safety 後連續呼叫兩次 `TurnOnThreadSafety()`
- **THEN** 只有第一次記錄一筆 Info

### Requirement: The off switch says what it turns off

The doc comment of `Config.Dangerously_TurnOffThreadSafety` SHALL name `AtomicDo` and `AtomicDoAll` as what stops locking, SHALL say that `Config`, the error buffer and logging stay safe, and SHALL name `TurnOnThreadSafety` as the way back.

#### Scenario: Reading the doc comment
- **WHEN** 讀取 `Dangerously_TurnOffThreadSafety` 的 doc comment
- **THEN** 註解提到 `AtomicDo`、`AtomicDoAll`、仍然安全的部分與 `TurnOnThreadSafety`
