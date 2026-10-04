# Tasks: cli-env-one-name

## 1. 測試先行

- [x] 1.1 `cli/commands/command_flags_test.go`：`TestEnvAndAccelOneShotFlagsAsBefore` 釘住 `env`、`accel` 旗標傳給 Run 的方式與說明文字，在舊程式上通過；`TestRegisteredFlagsReachRun` 在舊程式上編譯失敗；`TestBuildCobraCommandsNamesNoCommand` 在舊程式上失敗
- [x] 1.2 `cli/env/deprecated_wrappers_test.go`：26 個包裝函式都有 `Deprecated: use Default().<Name> instead.`，且仍作用在 `Default()` 上；在舊程式上失敗

## 2. 實作

- [x] 2.1 `cli/commands/registry.go`：`CommandHandler.Flags` 與 `CommandFlag`；`BuildCobraCommands` 依宣告註冊並轉交旗標，不再比對命令名稱
- [x] 2.1a 自我審查後：`Register` 拒絕沒有名稱或重複宣告的旗標，否則 `BuildCobraCommands` 會在 pflag 裡 panic；`TestRegisterRefusesAFlagTheShellCannotTake` 先失敗再通過
- [x] 2.2 `cli/commands/env.go`、`cli/commands/accel.go`：在註冊時宣告各自的旗標
- [x] 2.3 `cli/env`：26 個包裝函式加上 Deprecated 說明，`Default` 與 `SetBasePath` 的說明不再指向包裝函式
- [x] 2.4 以 `gofmt -r` 把 repo 內所有呼叫改成 `Default().X`（含以別名 `clienv` 匯入的檔案）
- [x] 2.5 `accel` 讀預設模式時改用 session 的 `ExecContext.Env`，沒有才用 `Default()`；`TestAccelModeComesFromTheSessionManager` 先在舊程式上失敗（編譯不過），修正後通過

## 3. 文件與紀錄

- [x] 3.1 `Docs/cli-dsl.md`：說明套件層函式已 Deprecated；`cli/AGENTS.md`：命令以 `Flags` 宣告旗標
- [x] 3.2 `CHANGELOG.md`、`CHANGELOG_TW.md`：`### CLI` 段落末尾新增條目
- [x] 3.3 `api-review.md`：CL-3 標明已處理的部分與留下的部分
- [x] 3.4 `AGENTS.md`：follow-up 記錄下一版移除包裝函式
- [x] 3.5 `delivery-status.md`：Latest Milestones 最上方新增條目
- [x] 3.6 skills：不需要修改，skills 不列函式，文件位置也沒變

## 4. 驗證

- [x] 4.1 `gofmt -s -l`、`go build ./...`、`go vet ./...`、`go test ./...`、`golangci-lint run`（0 issues）、`openspec validate cli-env-one-name --strict`
