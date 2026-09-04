# datafetch-twstock Specification

## Purpose
`datafetch` 對證交所（TWSE）與櫃買中心（TPEx）免登入端點的客戶端：逐月分頁的個股日線、當日三大法人買賣超、融資融券餘額、全市場日行情，輸出型別化的 `DataTable`（民國日期轉 `time.Time`、佔位符轉 nil）；含節流、重試、`Auto` 市場 fallback，測試以錄製的 fixture 回放、live 存取需 opt-in。

## Requirements
### Requirement: TWStock client configuration and transport

`datafetch` SHALL 提供 `TWStockConfig{Timeout, Interval, UserAgent string, Retries int, RetryBackoff time.Duration, Concurrency int}` 與 `TWStock(cfg TWStockConfig) (*twStock, error)`，零值欄位 SHALL 套用與 `YFinanceConfig.normalize` 相同型態的預設。每次請求 SHALL 遵守 `Interval` 節流；HTTP 非 2xx、逾時、JSON 解析失敗、或 payload `stat`／`tables` 表示無資料以外的錯誤 SHALL 依 `Retries` 與 `RetryBackoff` 重試後回錯。`TWMarket` SHALL 為 `TWMarketTWSE`、`TWMarketTPEx`、`TWMarketAuto`；`Auto` SHALL 先查 TWSE，收到「查無資料」時再查 TPEx。

#### Scenario: Throttle is honoured

- **WHEN** `Interval: 200ms`，連續發出兩個請求
- **THEN** 第二個請求的送出時間距第一個至少 200ms

#### Scenario: Retry then fail

- **WHEN** 伺服器連續回 500 且 `Retries: 2`
- **THEN** 共送出 3 次請求後回傳含狀態碼的錯誤

#### Scenario: Auto market falls through to TPEx

- **WHEN** 以 `TWMarketAuto` 查一檔上櫃股票的日線
- **THEN** TWSE 回「查無資料」後改查 TPEx 並回傳資料

### Requirement: Daily prices by stock and date range

`twStock` SHALL 提供 `DailyPrices(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error)`，逐月呼叫 TWSE `rwd/zh/afterTrading/STOCK_DAY`（`date=YYYYMM01&stockNo=`）或 TPEx `www/zh-tw/afterTrading/tradingStock`（`code=&date=YYYY/MM/01`），合併後只保留 `[from, to]` 內的列並依日期遞增。欄位 SHALL 為 `Date`（`time.Time`，由民國年 `115/09/01` 轉換）、`Code`、`Volume`（股數，`int64`）、`Turnover`（`int64`）、`Open`、`High`、`Low`、`Close`、`Change`（`float64`）、`Transactions`（`int64`）。含逗號的數字 SHALL 去逗號後轉型；`--`、空字串、`X` 等非數值 SHALL 轉為 nil。`from > to`、空 `code` SHALL 回錯。

#### Scenario: Two-month range yields two requests and typed columns

- **WHEN** 以固定 fixture 模擬 2026-08 與 2026-09 的 TWSE 回應，查 `2330` 從 2026-08-15 到 2026-09-03
- **THEN** 送出兩次請求，輸出只含該區間的日期，`Date` 為 `time.Time`，`Close` 為 `float64`，`Volume` 為 `int64`

#### Scenario: TPEx daily fixture parses the same schema

- **WHEN** 以 fixture 模擬 TPEx `tradingStock` 回應查 `6488`
- **THEN** 輸出欄位與 TWSE 相同，數值型別相同

#### Scenario: Non-numeric cells become nil

- **WHEN** fixture 中某列的 `Change` 為 `X` 或價格為 `--`
- **THEN** 該格為 nil，其餘欄位正常

### Requirement: Institutional trades, margin balances, and full daily quotes

`twStock` SHALL 提供 `InstitutionalTrades(date time.Time, market TWMarket)`（TWSE `rwd/zh/fund/T86?selectType=ALL`、TPEx `www/zh-tw/insti/dailyTrade?type=Daily`）、`MarginBalance(date time.Time, market TWMarket)`（TWSE `rwd/zh/marginTrading/MI_MARGN?selectType=ALL` 的個股表、TPEx 對應 margin 端點）、`AllDailyQuotes(market TWMarket)`（TWSE OpenAPI `exchangeReport/STOCK_DAY_ALL`、TPEx OpenAPI `tpex_mainboard_daily_close_quotes`），各回傳 `(*insyra.DataTable, error)`。三大法人表 SHALL 至少含 `Date`、`Code`、`Name`、`ForeignNet`、`TrustNet`、`DealerNet`、`TotalNet`（股數，`int64`）；融資融券表 SHALL 至少含 `Date`、`Code`、`Name`、`MarginBalance`、`ShortBalance`（`int64`）；全市場日行情 SHALL 含 `Date`、`Code`、`Name`、`Volume`、`Turnover`、`Open`、`High`、`Low`、`Close`、`Change`、`Transactions`。中文表頭 SHALL 在套件內對應到上述英文欄名；對應不到必要欄位 SHALL 回錯指出缺哪一欄。

#### Scenario: Institutional fixture maps headers

- **WHEN** 以 fixture 模擬 TWSE `T86` 回應
- **THEN** 輸出含 `ForeignNet`、`TrustNet`、`DealerNet`、`TotalNet` 四欄，型別 `int64`，`Date` 等於查詢日

#### Scenario: Missing required header is refused

- **WHEN** fixture 的表頭少了「證券代號」
- **THEN** 回傳指出該欄位名稱的錯誤

#### Scenario: Full quote table from OpenAPI

- **WHEN** 以 fixture 模擬 `STOCK_DAY_ALL`（含 `Date: "1150903"`）
- **THEN** 每列 `Date` 為 2026-09-03，價格為 `float64`，列數等於 fixture 筆數

### Requirement: Live access is opt-in

套件測試 SHALL 只在 `INSYRA_RUN_LIVE_TWSTOCK=1` 時對交易所發出真實請求；未設定時 live 測試 SHALL `Skip`。fixture 測試 SHALL 透過可注入的 `http.RoundTripper`（回放 `datafetch/testdata/twstock/` 的錄製回應）進行，不觸網也不開 socket。

#### Scenario: Live test skipped by default

- **WHEN** 未設定 `INSYRA_RUN_LIVE_TWSTOCK`
- **THEN** `go test ./datafetch/...` 不發出對 twse.com.tw 或 tpex.org.tw 的請求

