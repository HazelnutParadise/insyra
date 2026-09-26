## MODIFIED Requirements

### Requirement: TWStock client configuration and transport

`datafetch` SHALL 提供 `TWStockConfig{Timeout, Interval, UserAgent string, Retries int, RetryBackoff time.Duration, Concurrency int}` 與 `TWStock(cfg TWStockConfig) (*twStock, error)`，零值欄位 SHALL 套用與 `YFinanceConfig.normalize` 相同型態的預設。每次請求 SHALL 遵守 `Interval` 節流：相鄰兩個請求排定的開始時間至少相隔 `Interval`，且每個請求都不會早於排定的時間開始。客戶端在排定之後、送出之前的處理時間不在這個保證之內。HTTP 非 2xx、逾時、JSON 解析失敗、或 payload `stat`／`tables` 表示無資料以外的錯誤 SHALL 依 `Retries` 與 `RetryBackoff` 重試後回錯。`TWMarket` SHALL 為 `TWMarketTWSE`、`TWMarketTPEx`、`TWMarketAuto`；`Auto` SHALL 先查 TWSE，收到「查無資料」時再查 TPEx。

#### Scenario: Throttle is honoured

- **WHEN** `Interval: 200ms`，在時間 t0 之後連續發出兩個請求
- **THEN** 第二個請求不會在 t0 之後 200ms 以內送出

#### Scenario: Retry then fail

- **WHEN** 伺服器連續回 500 且 `Retries: 2`
- **THEN** 共送出 3 次請求後回傳含狀態碼的錯誤

#### Scenario: Auto market falls through to TPEx

- **WHEN** 以 `TWMarketAuto` 查一檔上櫃股票的日線
- **THEN** TWSE 回「查無資料」後改查 TPEx 並回傳資料
