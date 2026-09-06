# cli-env-name Specification

## Purpose
Environment names are validated against `^[A-Za-z0-9][A-Za-z0-9._-]*$` (no `..`) before being joined onto the envs directory, closing the path-traversal hole.

## Requirements
### Requirement: Environment names cannot escape the envs directory

`Manager.ResolveEnvPath`（因此 `Create`／`Open`／`Delete`／`Rename`／`Clear`／`Export`／`Import`）SHALL 拒絕含路徑分隔符、`..`，或不符合 `^[A-Za-z0-9][A-Za-z0-9._-]*$` 的名稱，錯誤訊息 SHALL 說明規則。

#### Scenario: Traversal is refused
- **WHEN** `Create("../outside")`
- **THEN** 回傳錯誤，檔案系統無變化

#### Scenario: Normal names still work
- **WHEN** `Create("proj-1.test")`
- **THEN** 成功

