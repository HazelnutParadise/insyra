# cli-registry-safety Specification

## Purpose
CLI 命令登錄表的並行安全：嵌入端可從多個 goroutine 同時註冊與分派命令，`Register`、`Dispatch` 與 `BuildCobraCommands` 都經同一把讀寫鎖存取登錄表。

## Requirements
### Requirement: Registry is safe for concurrent use

`commands.Register` 與 `commands.Dispatch` 並行呼叫在 race detector 下 SHALL NOT 回報 data race。

#### Scenario: Concurrent register and dispatch
- **WHEN** 多個 goroutine 同時 Register 不同名稱並 Dispatch
- **THEN** 無 race 回報且全部成功
