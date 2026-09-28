## MODIFIED Requirements

### Requirement: Every command is documented with its real usage

For every command in the CLI registry, `Docs/cli-dsl.md`'s command index SHALL have an entry whose usage line equals the command's `Usage` string. A documented entry that is not a registered command SHALL be allowed only for Cobra's `completion`. A test SHALL enforce this and SHALL fail rather than pass when it parses implausibly few entries.

#### Scenario: A command gains an option
- **WHEN** 某個指令的 `Usage` 新增了一個選項，但 `Docs/cli-dsl.md` 的指令索引沒有跟著改
- **THEN** 測試失敗並指出是哪個指令

#### Scenario: A command is added
- **WHEN** 新增一個指令但沒有寫進 `Docs/cli-dsl.md` 的指令索引
- **THEN** 測試失敗並指出缺少的指令

### Requirement: Every command appears in the topic lists

`Docs/cli-dsl.md`'s command groups SHALL name every registered command. A test SHALL enforce this and SHALL fail rather than pass when it finds implausibly few names.

#### Scenario: A command is missing from a topic list
- **WHEN** 新增的指令沒有出現在 `Docs/cli-dsl.md` 的 Command Groups
- **THEN** 測試失敗並指出缺少哪個指令
