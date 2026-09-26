# Tasks: cli-declared-arg-limits

## 1. Tests first

- [x] 1.1 `arg_limit_test.go`：每個已註冊指令都宣告 `Args`；`MaxArgs`、`WithAlias`、`FormArgs`、`FormArgsAt`、`OpenArgs` 各自擋下或放行的情形
- [x] 1.2 `arg_limit_dispatch_test.go`：經由 `Dispatch` 走真實入口，八個多餘引數的呼叫都回錯並指出該引數，五個正常呼叫照常執行（先紅：拿掉註冊時的檢查，八個全部被接受）

## 2. Implementation

- [x] 2.1 `arg_limit.go`：`ArgLimit` 與 `MaxArgs`、`FormArgs`、`FormArgsAt`、`OpenArgs`、`WithAlias`
- [x] 2.2 `registry.go`：`CommandHandler.Args`，`Register` 包裝 `Run` 先做數量檢查
- [x] 2.3 117 個指令逐一依 Usage 與實際解析宣告 `Args`

## 3. Docs, changelog, follow-ups

- [x] 3.1 `cli/AGENTS.md`：說明 `Args` 的四種宣告與何時用 `OpenArgs`
- [x] 3.2 `Docs/cli-dsl.md` 與 `skills/use-insyra-cli/SKILL.md`：多餘引數會被拒絕，`as <var>` 只給會存結果的指令
- [x] 3.3 `CHANGELOG.md`／`CHANGELOG_TW.md`：CLI 段新增 BREAKING
- [x] 3.4 `AGENTS.md`：移除「53 個指令接受多餘引數」follow-up，把盤點時發現、數量檢查抓不到的問題另列一條
- [x] 3.5 `delivery-status.md`

## 4. Verification

- [x] 4.1 文件與技能裡的指令範例逐行跑過宣告的上限，沒有一個被誤擋
- [x] 4.2 隔離的 HOME 實跑：one-shot 下 `iqr x junk` 回錯、exit 1，`iqr x` 照常；`.isr` 腳本裡同一行回錯後繼續下一行；`set dt 0 A -- -2.5` 照常
- [x] 4.3 `go test ./...`、`golangci-lint run`、`openspec validate cli-declared-arg-limits --strict`
