## ADDED Requirements

### Requirement: CI pins the versions it runs

每個 workflow 對同一個 action SHALL 使用同一個主版本；linter SHALL 釘在明確版號，SHALL NOT 使用 `latest`，否則 lint 結果無法重現。

#### Scenario: Two workflows using the same action
- **WHEN** 比較任兩個 workflow 的 `actions/checkout` 與 `actions/setup-go`
- **THEN** 版本相同
