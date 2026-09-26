## ADDED Requirements

### Requirement: The registry is read through a locked accessor

讀取 `Registry` SHALL 經過取用 `registryMu` 的存取函式（`LookupCommand`、`SnapshotRegistry`），SHALL NOT 直接走訪 map。嵌入者可以在任何時候註冊指令。

#### Scenario: help and tab completion
- **WHEN** `help` 或 REPL 的補全列出指令
- **THEN** 它們透過加鎖的快照讀取，而不是直接走訪 `Registry`

### Requirement: The help table lines up

`help` 的指令清單 SHALL 依最長的指令名稱決定欄寬，SHALL NOT 使用固定寬度。

#### Scenario: A command name longer than twelve characters
- **WHEN** 清單中含 `knn_neighbors`
- **THEN** 所有描述都從同一欄開始
