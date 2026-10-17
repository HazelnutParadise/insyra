# cli-negative-literals Specification

## Purpose
`newdl`、`addrow`、`addcol` 關閉 flag 解析，使一次性 CLI 模式接受負數字面值；`help <cmd>` 仍可用。

## Requirements

### Requirement: Value-taking commands accept negative literals in one-shot mode

`newdl`、`addrow`、`addcol` SHALL 設定 `DisableFlagParsing`，使 `insyra newdl 0.01 -0.004 as r` 在一次性 CLI 模式下把 `-0.004` 當成值。REPL 與 `.isr` 的行為 SHALL 不變。`help newdl` 等 SHALL 仍可用。`Docs/cli-dsl.md` SHALL 說明其他接受值的命令可用 `--` 分隔。

#### Scenario: Negative literal in one-shot mode

- **WHEN** 透過 `BuildCobraCommands` 執行 `newdl 0.01 -0.004 0.02 as r`
- **THEN** `r` 為含三個值的 `DataList`，第二個為 −0.004

#### Scenario: Help still renders

- **WHEN** 執行 `help newdl`
- **THEN** 顯示 Usage，不報未知 flag

### Requirement: Single-sample tests accept a negative mu in one-shot mode

`wilcoxon`, `ttest` and `ztest` SHALL disable flag parsing, so that in one-shot CLI mode a negative value for `mu`, such as `-1` in `ttest single x -1`, is read as the value and not as a flag.

#### Scenario: Negative mu for a t-test
- **WHEN** `ttest single x -1` is run in one-shot mode against a variable `x`
- **THEN** the command runs with `mu` equal to -1 and returns no error

#### Scenario: Negative mu for a Z-test with a sigma
- **WHEN** `ztest single x -1 2` is run in one-shot mode
- **THEN** the command runs with `mu` equal to -1 and `sigma` equal to 2 and returns no error
