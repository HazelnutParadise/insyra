## ADDED Requirements

### Requirement: Ex-rights and ex-dividend table

`twStock` SHALL 提供 `ExRights(from, to time.Time, market TWMarket) (*insyra.DataTable, error)`。TWSE SHALL 呼叫 `rwd/zh/exRight/TWT49U?startDate=YYYYMMDD&endDate=YYYYMMDD&response=json`，超過一年的區間 SHALL 切成多次請求後合併。輸出欄位 SHALL 為 `Date`（除權息日，`time.Time`，由 `115年06月01日` 格式轉換）、`Code`、`Name`、`PrevClose`（除權息前收盤價）、`RefPrice`（除權息參考價）、`Distribution`（權值+息值）、`Kind`（權/息 欄：`息`→`"dividend"`、`權`→`"rights"`、`權息`→`"both"`）、`AdjFactor`（`RefPrice / PrevClose`，`float64`）。列 SHALL 依 `Date` 遞增，再依 `Code`。`from > to` SHALL 回錯。TPEx SHALL 使用櫃買中心對應的除權除息計算結果表端點；實作時若無法確認該端點，`TWMarketTPEx` SHALL 回傳明確指出「TPEx ex-rights not supported」的錯誤，`TWMarketAuto` SHALL 只查 TWSE。

#### Scenario: TWSE fixture parses into typed columns

- **WHEN** 以 fixture 回放 `TWT49U` 查 2026-06-01 到 2026-09-03
- **THEN** 第一列 `Code == "2612"`、`Date` 為 2026-06-01、`PrevClose == 57.7`、`RefPrice == 55.5`、`Distribution == 2.2`、`Kind == "dividend"`、`AdjFactor` 等於 `55.5/57.7`（1e-12 內）

#### Scenario: Ranges longer than a year are paged

- **WHEN** 查 2024-01-01 到 2026-09-03
- **THEN** 送出三次請求，區間分別不超過一年且首尾相接，結果合併後遞增

#### Scenario: Missing header is refused

- **WHEN** fixture 表頭缺「除權息參考價」
- **THEN** 回傳指出該欄位名稱的錯誤

#### Scenario: TPEx without a confirmed endpoint is explicit

- **WHEN** TPEx 端點未實作而以 `TWMarketTPEx` 呼叫
- **THEN** 回傳訊息含 `not supported` 的錯誤，不回空表

### Requirement: Backward-adjusted daily prices

`twStock` SHALL 提供 `DailyPricesAdjusted(code string, from, to time.Time, market TWMarket) (*insyra.DataTable, error)`，輸出 SHALL 含 `DailyPrices` 的全部欄位，再加 `AdjFactor`、`AdjOpen`、`AdjHigh`、`AdjLow`、`AdjClose`（`float64`，原價為 nil 時對應調整價亦為 nil）。調整 SHALL 為向後調整：對區間內該檔股票的每個除權息日 `d`（取自 `ExRights(from, to, market)` 中 `Code` 相符的列），所有 `Date < d` 的列 SHALL 乘上該日的 `AdjFactor`；多個除權息日的因子 SHALL 累乘；最後一個除權息日當天及之後的列 `AdjFactor == 1`。區間內沒有除權息日時，調整價 SHALL 等於原價。`to` 之後的除權息不納入。

#### Scenario: One ex-dividend day scales earlier bars

- **WHEN** fixture 提供 2330 在 2026-08-15 的日線與一筆 2026-08-15 除息（`PrevClose 100`、`RefPrice 98`）
- **THEN** 2026-08-15 之前的列 `AdjFactor == 0.98`、`AdjClose == Close × 0.98`，2026-08-15 起 `AdjFactor == 1`、`AdjClose == Close`

#### Scenario: Two ex-dates compound

- **WHEN** 區間內有兩個除權息日，因子分別為 0.98 與 0.95
- **THEN** 第一個除權息日之前的列 `AdjFactor == 0.98 × 0.95`，兩日之間為 0.95，第二日起為 1

#### Scenario: No ex-date leaves prices unchanged

- **WHEN** 區間內 `ExRights` 沒有該 `Code` 的列
- **THEN** 每列 `AdjFactor == 1` 且 `AdjClose == Close`

#### Scenario: Return series has no artificial drop

- **WHEN** 以 fixture 的除息日前後兩筆收盤價（前一日 100、除息日 98、除息 2 元）計算
- **THEN** `AdjClose` 的日報酬在除息日為 0（1e-12 內），而原始 `Close` 的日報酬為 −2%

#### Scenario: Nil prices stay nil

- **WHEN** 某列 `Open` 為 nil
- **THEN** 該列 `AdjOpen` 為 nil，其他調整欄照常

### Requirement: ROC long date format

`parseROCDate` SHALL 額外接受 `115年06月01日` 格式並轉為 2026-06-01；不合法的年月日 SHALL 回錯。

#### Scenario: Long format parses

- **WHEN** 輸入 `115年06月01日`
- **THEN** 回傳 2026-06-01 UTC

#### Scenario: Live access remains opt-in

- **WHEN** 設定 `INSYRA_RUN_LIVE_TWSTOCK=1`
- **THEN** live 測試對 TWSE 各呼叫一次 `ExRights` 與 `DailyPricesAdjusted` 並檢查非空與型別；未設定時 Skip
