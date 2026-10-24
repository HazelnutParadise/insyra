# Tasks: engine-dsl-manager

## 1. 測試

- [x] 1.1 `engine/dsl/manager_test.go`：只用 `engine/dsl` 建立 Manager、建立環境、開 session 並讀回狀態（`TestNewManagerCreatesAndManagesSessionEnvironment`）

## 2. 實作

- [x] 2.1 `engine/dsl/manager.go`：`Manager`、`NewManager`、`DefaultManager` 與五個型別，型別為 `cli/env` 同名型別的別名

## 3. 文件與紀錄

- [x] 3.1 `Docs/cli-dsl.md`：`engine/dsl` 範例改用 `NewManager` 與 `DefaultManager`
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### CLI` 段落末尾新增條目
