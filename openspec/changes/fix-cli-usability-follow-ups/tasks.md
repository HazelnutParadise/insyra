# Tasks: fix-cli-usability-follow-ups

## 1. Tests first

- [ ] 1.1 `datalist_dates_test.go`：ISO 字串、自訂 layout、已是 `time.Time`、不可解析為 nil；`DataTable.ParseDatesCols` 就地轉換後 `Resample` 可用；找不到的欄位 warning
- [ ] 1.2 `datatable_from_sql_test.go`：既有 `ParseDates` 測試不改即通過
- [ ] 1.3 `cli/commands/parsedates_test.go`：DataList 整列、DataTable `cols`、`layout` 重複、缺 `cols` 回錯、`as`／`$result`；`.isr` 端到端 `load csv → parsedates → resample`
- [ ] 1.4 `cli/env/state_test.go`：float 與 int64 round trip；字串／布林不變
- [ ] 1.5 `cli/commands/newdl_test.go`（或既有檔）：經 `BuildCobraCommands` 執行 `newdl 0.01 -0.004 0.02 as r`；`addrow`、`addcol` 各一個負數案例；`help newdl` 正常

## 2. Implementation

- [ ] 2.1 `datalist_dates.go`：`ParseDates`；`datatable_from_sql.go` 的 `dateParseLayouts` 改為共用預設並讓 `ReadSQL` 走 `ParseDatesCols`；`datatable.go`／新檔加 `ParseDatesCols`；`interfaces.go` 加兩個方法
- [ ] 2.2 `cli/commands/parsedates.go`：註冊與實作
- [ ] 2.3 `cli/env/state.go`：`LoadState` 對頂層純量套 `coerceEnvNumber`
- [ ] 2.4 `newdl.go`、`addrow.go`、`addcol.go`：`DisableFlagParsing: true`

## 3. Docs, changelog, skills, follow-ups

- [ ] 3.1 `Docs/DataList.md` 加 `ParseDates`；`Docs/DataTable.md` 加 `ParseDatesCols`；`Docs/cli-dsl.md` 加 `parsedates`（Command Groups、Index、B6 前置步驟）與 `--` 說明
- [ ] 3.2 `skills/insyra/SKILL.md`、`skills/use-insyra-cli/SKILL.md` 與 references 同步
- [ ] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md`：`### Core` 一條（ParseDates）、`### CLI` 一條（parsedates、純量 round trip、負數字面值）
- [ ] 3.4 `AGENTS.md`：刪除三條 2026-09-05 的 follow-up

## 4. Verification

- [ ] 4.1 `go test ./...` 全綠；`golangci-lint run` 無新錯誤；本機一次性執行 `go run ./cmd/insyra newdl 0.01 -0.004 as r` 成功
- [ ] 4.2 `openspec validate fix-cli-usability-follow-ups --strict` 通過
