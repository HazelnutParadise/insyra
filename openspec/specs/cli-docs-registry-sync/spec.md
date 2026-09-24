# cli-docs-registry-sync Specification

## Purpose
Keeps the CLI documentation true to the commands: every command's usage line in the three usage documents equals its `help` text, both topic lists name every command, and a command's `Usage` names every option it parses.

## Requirements

### Requirement: Every command is documented with its real usage

For every command in the CLI registry, `Docs/cli-dsl.md`'s command index, `cli-command-usage.md` and `cli-command-guide.md` SHALL each have an entry whose usage line equals the command's `Usage` string. A documented entry that is not a registered command SHALL be allowed only for Cobra's `completion` and the guide's `load sql` and `save sql` sub-sections. A test SHALL enforce this and SHALL fail rather than pass when it parses implausibly few entries.

#### Scenario: A command gains an option
- **WHEN** 某個指令的 `Usage` 新增了一個選項，但文件沒有跟著改
- **THEN** 測試失敗並指出是哪份文件的哪個指令

#### Scenario: A command is added
- **WHEN** 新增一個指令但沒有寫進三份文件
- **THEN** 測試失敗並指出缺少的文件

### Requirement: Every command appears in the topic lists

`Docs/cli-dsl.md`'s command groups and `skills/use-insyra-cli/references/cli-commands.md` SHALL name every registered command. A test SHALL enforce this and SHALL fail rather than pass when it finds implausibly few names.

#### Scenario: A command is missing from a topic list
- **WHEN** 新增的指令沒有出現在 Command Groups 或 `cli-commands.md`
- **THEN** 測試失敗並指出是哪一份清單缺少哪個指令

### Requirement: A command's Usage names every option it accepts

A command's `Usage` string SHALL list every argument and option the command parses, and SHALL NOT mark a required argument as optional. A command's usage error SHALL show the same shape as its `Usage`.

#### Scenario: A command that stores its result
- **WHEN** 執行 `help pca` 或 `help regression`
- **THEN** Usage 包含 `[as <var>]`

#### Scenario: A required value
- **WHEN** 執行 `help count`，或不帶參數執行 `count`
- **THEN** Usage 與用法錯誤都寫成 `count <var> <value>`，不把 value 標成選填
