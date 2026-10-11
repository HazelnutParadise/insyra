# Tasks: engine-dsl-register-command

## 1. 測試

- [x] 1.1 `engine/dsl/commands_test.go`：只用 `engine/dsl` 註冊命令並在 session 執行、輸出與存檔、參數數量檢查、名稱重複與缺 `Run` 的拒絕、命令錯誤傳回 `Execute`（`TestRegisterACommandAndRunItInASession`、`TestRegisterRefusesTakenNamesAndInvalidHandlers`、`TestRegisteredCommandErrorsReachSessionCaller`）
- [x] 1.2 `cli/commands/registry_test.go`：負數參數上限被拒絕（`TestRegisterRejectsNegativeArgumentLimits`）；重複註冊不改動原本的命令（`TestDuplicateRegistrationDoesNotMutateOriginalHandler`）

## 2. 實作

- [x] 2.1 `engine/dsl/commands.go`：`Register`、`CommandHandler`、`ExecContext`、`CommandFlag`、`ArgLimit`、`MaxArgs`、`FormArgs`、`FormArgsAt`、`OpenArgs`
- [x] 2.2 `cli/commands/registry.go`：註冊時先檢查名稱，再包住 `Run` 並保存自己的副本；`cli/commands/arg_limit.go`：`validate` 拒絕負數上限與負數形式位置
- [x] 2.3 `Docs/cli-dsl.md`：新增「Register a command」範例，並說明 `ArgLimit` 只檢查參數數量

## 3. 紀錄

- [x] 3.1 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### CLI` 段落末尾新增條目
