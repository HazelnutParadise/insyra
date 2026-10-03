## ADDED Requirements

### Requirement: A one-shot flag may belong to several forms
`CommandFlag.Form` MAY 以 `|` 分隔列出多個字（例如 `"import|delete"`），寫法與 Usage 的選項相同；第一個參數等於其中任一個字（不分大小寫）時，`BuildCobraCommands` SHALL 把這個旗標交給 `Run`，否則 SHALL 丟掉它。

#### Scenario: --force reaches env delete
- **WHEN** the shell runs `insyra --env work env delete default --force`
- **THEN** `env`'s Run receives `delete default --force`

#### Scenario: --force is still dropped from env clear
- **WHEN** the shell runs `insyra env clear a --force`
- **THEN** `env`'s Run receives `clear a`
