# core-multilock-reentry Specification

## Purpose
`AtomicDoAll` 在 `AtomicDo` 內再入的行為：內聯執行而非死鎖。

## Requirements
### Requirement: AtomicDoAll inside AtomicDo does not deadlock

當呼叫 goroutine 已持有任一實例的鎖時，`AtomicDoAll`／`AtomicDoN` SHALL 內聯執行回呼而不再加鎖，並觸發 trust-zone hook；兩個 goroutine 互為鏡像的巢狀呼叫 SHALL 在有限時間內完成。

#### Scenario: Mirror-image nesting
- **WHEN** G1 在 `a.AtomicDo` 內呼叫 `AtomicDoAll(f, a, b)`，G2 同時在 `b.AtomicDo` 內呼叫 `AtomicDoAll(f, a, b)`
- **THEN** 兩者都在 3 秒內返回

