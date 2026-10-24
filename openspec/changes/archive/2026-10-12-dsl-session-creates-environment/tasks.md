# Tasks: dsl-session-creates-environment

## 1. 測試

- [x] 1.1 `engine/dsl/manager_test.go`：兩個 goroutine 同時對同一個不存在的環境呼叫 `NewSession` 都成功（`TestConcurrentNewSessionsCreateOneEnvironmentSafely`）；環境資料夾存在但不完整時回傳錯誤（`TestNewSessionRejectsIncompleteEnvironment`）
- [x] 1.2 `cli/env/create_race_test.go`：並行 `Create` 只有一個成功（`TestConcurrentCreateHasOneWinner`）；既有檔案不被覆寫（`TestCreateDoesNotOverwriteAnExistingFile`）；並行 `EnsureDefaultEnvironment` 全部成功（`TestConcurrentEnsureDefaultEnvironment`）

## 2. 實作

- [x] 2.1 `engine/dsl/dsl.go`：`NewSession` 遇到不存在的環境就建立；建立失敗且別人沒有完成該環境時回傳錯誤
- [x] 2.2 `cli/env/manager.go`：`Create` 用 `os.Mkdir` 搶建立權（父層目錄仍用 `MkdirAll`）；`writeDefaultFiles` 只寫不存在的檔案；`EnsureDefaultEnvironment` 容許別人搶先建好

## 3. 文件與紀錄

- [x] 3.1 `Docs/cli-dsl.md`：說明 `NewSession` 會建立缺少的環境
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### CLI` 段落末尾新增條目
