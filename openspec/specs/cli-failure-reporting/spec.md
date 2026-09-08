# cli-failure-reporting Specification

## Purpose
CLI 指令做不到被交代的事時，必須以非零狀態結束並說明原因。

## Requirements
### Requirement: A command that fails exits non-zero

`sort`、`dropcol`、`droprow`、`swap`、`ccl`、`addcolccl`、`setcolnames`、`sample` 在目標不存在或程式庫記錄錯誤時 SHALL 回傳錯誤，SHALL NOT 印出成功訊息。錯誤訊息 SHALL 指出缺少的目標。

#### Scenario: Sorting by a column that is not there
- **WHEN** 對沒有 `nonexistent` 欄的表執行 `sort dt nonexistent`
- **THEN** 回傳錯誤，且輸出不含 "sorted"

#### Scenario: A CCL expression that cannot compile
- **WHEN** 執行 `ccl dt "bogus((("`
- **THEN** 回傳錯誤，且輸出不含 "ccl executed"

