# Tasks: add-cli-timeseries-commands

## 1. Tests first

- [ ] 1.1 `cli/commands/timeseries_test.go`：`ewm` 遞迴均值 golden、選項解析等於直接呼叫 `EWM`、衰減值 0 回錯、未知 reducer 回錯、非 DataList 回錯；`rolling ... beta <other>` 縮放 benchmark 為常數、`cov` 缺第二變數回錯、既有 reducer 仍可用
- [ ] 1.2 `cli/commands/resample_test.go`：月線 OHLCV 含 `:name` 改名；未知 op 列出可用運算子；`col:op:name:extra` 格式回錯；非 time 欄回錯含列號；`as` 與 `$result`

## 2. Implementation

- [ ] 2.1 `cli/commands/timeseries.go`：註冊 `ewm`（Usage、Forms、Examples、Run）；`rolling` 的 Forms 與 reducer switch 加 `cov`/`beta`
- [ ] 2.2 `cli/commands/resample.go`：註冊 `resample`，重用 `groupby` 的運算子對應

## 3. Docs, changelog & skills

- [ ] 3.1 `Docs/cli-dsl.md`：Command Groups「Time Series / Transforms」加 `ewm`、`resample`；Full Command Index 加兩列並更新 `rolling` 的 Usage；Quickstart 加一段 `resample` 月線範例
- [ ] 3.2 `skills/use-insyra-cli/SKILL.md` 與 `references/cli-commands.md`、`cli-command-guide.md`、`cli-command-usage.md` 同步新命令與 `rolling` 新形式
- [ ] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### CLI` 段末各加一條

## 4. Verification

- [ ] 4.1 `go test ./cli/...` 與 `go test ./...` 全綠；`golangci-lint run` 無新錯誤；`go run ./cmd/insyra help ewm`、`help rolling`、`help resample` 顯示新形式
- [ ] 4.2 `openspec validate add-cli-timeseries-commands --strict` 通過
