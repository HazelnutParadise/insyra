# Tasks: add-cli-fetch-tw

## 1. Tests first (cli/commands/fetch_tw_test.go)

- [ ] 1.1 假客戶端記錄呼叫：六個形式各一測試，檢查傳入的 code、日期、market 與存入的變數
- [ ] 1.2 錯誤測試：日期格式、`from > to`、未知 market、未知形式在請求前回錯且假客戶端未被呼叫；函式庫錯誤含 `fetch tw:` 前綴
- [ ] 1.3 工廠設定：預設 `Interval 300ms`、`Retries 2`；`config fetch.tw.interval_ms 1000` 後為 1s
- [ ] 1.4 既有 `fetch yahoo` 測試（若無則補一個最小的分派測試）確認不受影響

## 2. Implementation

- [ ] 2.1 `cli/commands/fetch_tw.go`：`twStockClient` 介面、`newTWStockClient` 工廠、日期／market 解析、六個形式
- [ ] 2.2 `cli/commands/fetch.go`：依 `args[0]` 分派 `yahoo`／`tw`；Usage、Forms、Examples 加 `tw` 形式
- [ ] 2.3 `config` 讀取 `fetch.tw.interval_ms`（依既有 config 機制，必要時登記 key）

## 3. Docs, changelog & skills

- [ ] 3.1 `Docs/cli-dsl.md`：`fetch` 的 Usage 與 Forms 加 `tw`；Full Command Index 更新 `fetch` 列；Quickstart 加「Taiwan stock beta from the CLI」（`fetch tw ... adjprices twse` → `col AdjClose` → `pctchange` → `clean nil` → `quant beta`）
- [ ] 3.2 `skills/use-insyra-cli/SKILL.md` 與三份 references 同步 `fetch tw`
- [ ] 3.3 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` `### CLI` 段末各加一條

## 4. Verification

- [ ] 4.1 `go test ./cli/...` 與 `go test ./...` 全綠且不觸網；`golangci-lint run` 無新錯誤；`go run ./cmd/insyra help fetch` 顯示 `tw` 形式；一次真實 `insyra fetch tw 2330 prices 2026-08-01 2026-08-31 twse` 在本機成功
- [ ] 4.2 `openspec validate add-cli-fetch-tw --strict` 通過
