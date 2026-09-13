# actor-close-semantics Specification

## Purpose
`DataList`／`DataTable` 的 `Close()` 語意：它停止的是加鎖，不是已在等鎖、已排隊的工作，排隊中的操作不得被靜默丟棄。

## Requirements
### Requirement: Close stops locking, not queued work

`Close()` 之後取得鎖的 `AtomicDo` 回呼 SHALL 仍然執行。在等待鎖期間發生的 `Close()` SHALL NOT 使該操作被靜默丟棄。

#### Scenario: Close races a queued Append
- **WHEN** 一個 goroutine 在 `AtomicDo` 內呼叫 `Close()`，另一個同時 `Append`
- **THEN** `Append` 生效，`Len()` 增加

