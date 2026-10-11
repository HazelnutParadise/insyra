# Tasks: cli-env-one-name

## 1. 測試

- [x] 1.1 `cli/commands/command_flags_test.go`：`TestEnvAndAccelOneShotFlagsAsBefore` 釘住 `env`、`accel` 旗標傳給 Run 的方式與說明文字；`TestRegisteredFlagsReachRun` 確認宣告的旗標會傳給 Run；`TestRegisterRefusesAFlagTheShellCannotTake` 確認 `Register` 拒絕不能用的旗標；`TestBuildCobraCommandsNamesNoCommand` 確認 builder 不點名命令
- [x] 1.2 `cli/env/deprecated_wrappers_test.go`：`TestPackageLevelWrappersAreDeprecated` 確認包裝函式都有 Deprecated 說明；`TestDeprecatedWrappersActOnDefault` 確認包裝函式仍作用在 `Default()` 上

## 2. 實作

- [x] 2.1 `cli/commands/registry.go`：`CommandHandler.Flags` 與 `CommandFlag`；`BuildCobraCommands` 依宣告註冊並轉交旗標，不再比對命令名稱；`Register` 拒絕沒有名稱或重複宣告的旗標
- [x] 2.2 `cli/commands/env.go`、`cli/commands/accel.go`：在註冊時宣告各自的旗標
- [x] 2.3 `cli/env/manager.go`、`cli/env/config.go`、`cli/env/state.go`：26 個包裝函式加上 `Deprecated:` 說明，指向 `Default()` 的對應方法
- [x] 2.4 `cli/env_roundtrip_test.go`、`cli/root_flags_test.go`、`cli/repl/api_test.go`、`engine/dsl/dsl_test.go` 改用 `Default()`
- [x] 2.5 `cli/commands/accel.go`：預設模式改從 session 的 `ExecContext.Env` 讀取，沒有才用 `Default()`

## 3. 紀錄

- [x] 3.1 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### CLI` 段落末尾新增條目
