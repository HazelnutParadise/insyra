# Tasks: cli-reject-ignored-args

## 1. Tests first

- [x] 1.1 `ignored_args_test.go`：九個統計指令單一引數照常執行，`x as m` 回錯且不存 `m`；`accel devices junk`、`accel plan --precision float32` 回錯（先紅：十一項全部被接受）
- [x] 1.2 `batch7_test.go`：Usage 不宣稱 `--precision`；Cobra 只註冊 `--mode`（原本的斷言要求宣稱並註冊一個沒有任何動作讀的旗標，改成相反）

## 2. Implementation

- [x] 2.1 `makeDLNumberPrinter` 帶指令名稱，多餘引數回錯並指出是哪一個
- [x] 2.2 `accel`：移除 `--precision` 的解析、Cobra 旗標與 Usage，其他引數回錯

## 3. Docs, changelog, follow-ups

- [x] 3.1 `Docs/cli-dsl.md` 與三份技能參考移除 `--precision`
- [x] 3.2 `CHANGELOG.md`／`CHANGELOG_TW.md`：batch 7 那條不再描述 `--precision`；新增 BREAKING
- [x] 3.3 `AGENTS.md`：follow-up 改成只列還沒處理的 53 個指令；`api-review.md` 的 CLI-10 註記 `--precision` 已移除
- [x] 3.4 `delivery-status.md`

## 4. Verification

- [x] 4.1 `go test ./...` 全綠、`-race` 跑 `cli/commands`、`golangci-lint run` 0 issues
- [x] 4.2 隔離的 HOME 實跑：one-shot 下 `mean x as m` 回錯且沒存 `m`、`accel devices junk` 回錯、`accel plan --precision float32` 被 Cobra 以 unknown flag 拒絕、`accel plan --mode cpu` 照常；`.isr` 腳本裡同一行也回錯
- [x] 4.3 `openspec validate cli-reject-ignored-args --strict`
