# cli-entry Specification

## Purpose
TBD - created by archiving change cli-repl. Update Purpose after archive.
## Requirements
### Requirement: CLI entry point with cobra
系統 SHALL 提供 `cmd/insyra/main.go` 作為 CLI 程式入口，使用 cobra 框架建立 root command。

#### Scenario: No arguments opens REPL
- **WHEN** 使用者執行 `insyra`（不帶子命令）
- **THEN** 系統進入 REPL 互動模式，使用預設環境（`default`）或上次使用的環境

#### Scenario: Subcommand execution
- **WHEN** 使用者執行 `insyra <command> <args...>`
- **THEN** 系統執行對應的命令並輸出結果

#### Scenario: Global flag --env
- **WHEN** 使用者執行 `insyra --env my-project <command> <args...>`
- **THEN** 系統在指定的環境 `my-project` 下執行命令（讀取/寫入該環境的 state）

#### Scenario: Global flag --no-color
- **WHEN** 使用者執行 `insyra --no-color <command>`
- **THEN** 系統輸出不帶 ANSI 色碼

#### Scenario: Global flag --log-level
- **WHEN** 使用者執行 `insyra --log-level debug <command>`
- **THEN** 系統設定 insyra 的 LogLevel 為 Debug

### Requirement: Version command
系統 SHALL 提供 `version` 命令，顯示 insyra 版本資訊。

#### Scenario: Display version
- **WHEN** 使用者執行 `insyra version`
- **THEN** 系統顯示 `insyra.Version` 和 `insyra.VersionName`（如 `insyra v0.2.14 (Pier-2)`）

### Requirement: Help command
系統 SHALL 提供 `help` 命令，顯示命令說明。

#### Scenario: General help
- **WHEN** 使用者執行 `insyra help`
- **THEN** 系統列出所有可用命令及簡短說明

#### Scenario: Command-specific help
- **WHEN** 使用者執行 `insyra help load`
- **THEN** 系統顯示 `load` 命令的詳細用法與範例

