# cli-fetch-tw Specification

## Purpose
CLI／REPL／DSL 的 `fetch tw` 來源：六個形式對應 `datafetch.TWStock` 的日線、還原日線、除權息表、三大法人、融資融券、全市場日行情；日期 `YYYY-MM-DD`、market 關鍵字、請求前驗證、函式庫錯誤加前綴回傳；客戶端經套件層工廠建立，節流可由 `config fetch.tw.interval_ms` 覆寫。

## Requirements
### Requirement: fetch tw forms

`fetch` SHALL 接受來源 `tw`，形式為 `fetch tw <code> prices <from> <to> [market]`、`fetch tw <code> adjprices <from> <to> [market]`、`fetch tw exrights <from> <to> [market]`、`fetch tw institutional <date> [market]`、`fetch tw margin <date> [market]`、`fetch tw quotes [market]`，各自呼叫 `TWStock` 的 `DailyPrices`、`DailyPricesAdjusted`、`ExRights`、`InstitutionalTrades`、`MarginBalance`、`AllDailyQuotes`。日期 SHALL 為 `YYYY-MM-DD`；`market` SHALL 為 `twse`、`tpex`、`auto`（預設 `auto`）並對應 `TWMarket`。結果 SHALL 存到 `as <var>` 或 `$result`。日期格式錯誤、`from > to`、未知 market、未知形式 SHALL 在發出請求前回傳含用法的錯誤；函式庫錯誤 SHALL 加 `fetch tw:` 前綴原樣回傳。既有 `fetch yahoo` 形式 SHALL 不變。

#### Scenario: Adjusted prices into a variable

- **WHEN** 以假客戶端執行 `fetch tw 2330 adjprices 2026-08-01 2026-09-03 as p`
- **THEN** 假客戶端收到 `DailyPricesAdjusted("2330", 2026-08-01, 2026-09-03, TWMarketAuto)`，`p` 為其回傳的 `DataTable`

#### Scenario: Market keyword maps to TWMarket

- **WHEN** 執行 `fetch tw 6488 prices 2026-08-01 2026-08-31 tpex`
- **THEN** 假客戶端收到 `TWMarketTPEx`

#### Scenario: Bad date is rejected before any request

- **WHEN** 執行 `fetch tw 2330 prices 2026/08/01 2026-09-03`
- **THEN** 回傳含日期格式的錯誤，假客戶端未被呼叫

#### Scenario: Library error is surfaced

- **WHEN** 假客戶端對 `exrights ... tpex` 回傳 not supported 錯誤
- **THEN** 命令錯誤以 `fetch tw:` 開頭並含原訊息

#### Scenario: Yahoo forms unchanged

- **WHEN** 執行既有的 `fetch yahoo AAPL quote as q`
- **THEN** 行為與變更前相同

### Requirement: Client factory and interval configuration

`fetch tw` SHALL 透過套件層級的工廠函式建立客戶端，測試可替換為假客戶端；真實工廠 SHALL 以 `Interval: 300ms`、`Retries: 2` 建立 `datafetch.TWStock`，並讀取 `config` 的 `fetch.tw.interval_ms` 覆寫 `Interval`。

#### Scenario: Interval from config

- **WHEN** `config fetch.tw.interval_ms 1000` 後執行任一 `fetch tw` 形式
- **THEN** 工廠收到 `Interval == 1s`

#### Scenario: Default interval

- **WHEN** 未設定該 config
- **THEN** 工廠收到 `Interval == 300ms`、`Retries == 2`

