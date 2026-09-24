## ADDED Requirements

### Requirement: Registry is safe for concurrent use

`commands.Register` 與 `commands.Dispatch` 並行呼叫在 race detector 下 SHALL NOT 回報 data race。

#### Scenario: Concurrent register and dispatch
- **WHEN** 多個 goroutine 同時 Register 不同名稱並 Dispatch
- **THEN** 無 race 回報且全部成功
