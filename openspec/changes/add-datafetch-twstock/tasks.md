# Tasks: add-datafetch-twstock

## 1. Fixtures and tests first

- [ ] 1.1 錄製 fixture 到 `datafetch/testdata/twstock/`：TWSE `STOCK_DAY`（2330，2026-08 與 2026-09）、TPEx `tradingStock`（6488，2026-08）、TWSE `T86`、TPEx `insti/dailyTrade`、TWSE `MI_MARGN`、TPEx margin、OpenAPI `STOCK_DAY_ALL` 與 TPEx `tpex_mainboard_daily_close_quotes`（各截取前數十列即可）、一份 TWSE「查無資料」回應
- [ ] 1.2 `datafetch/twstock_parse_test.go`：`parseROCDate`（`115/09/01`、`1150903`）、`parseNumber`（逗號、`--`、`X`、空字串、負數）、表頭對應缺欄回錯
- [ ] 1.3 `datafetch/twstock_test.go`（`httptest.Server`）：節流間隔；重試次數與最終錯誤；`Auto` 由 TWSE 落到 TPEx；`DailyPrices` 兩個月兩次請求、區間過濾、型別；TPEx 日線同 schema；三大法人四欄；融資融券兩欄；`AllDailyQuotes` 日期與列數；`from > to`、空 code 回錯
- [ ] 1.4 `datafetch/twstock_live_test.go`：`INSYRA_RUN_LIVE_TWSTOCK=1` 才跑，各方法各打一次並檢查非空與型別

## 2. Implementation

- [ ] 2.1 `datafetch/twstock_parse.go`：`parseROCDate`、`parseNumber`、各表的中文表頭→英文欄名對應與必要欄檢查
- [ ] 2.2 `datafetch/twstock.go`：`TWStockConfig.normalize`、`TWMarket`、`TWStock` 建構子（可注入 base URL）、節流／重試／逾時、`doJSON`
- [ ] 2.3 `DailyPrices`：逐月分頁、TWSE／TPEx 兩種 payload 解析、`Auto` 落下、區間過濾、遞增排序、`Market` 欄
- [ ] 2.4 `InstitutionalTrades`、`MarginBalance`、`AllDailyQuotes`
- [ ] 2.5 `datafetch/init.go` 或套件 doc comment 加一句

## 3. Docs, changelog & skills

- [ ] 3.1 `Docs/datafetch.md`：新增「Taiwan Stock Exchanges (TWStock)」章節（Quick Start、設定、四個方法與欄位表、資料授權連結、分頁與節流建議、限制）
- [ ] 3.2 `Docs/README.md`、`README.md`、`README_TW.md` 的 datafetch 列加入 TWSE／TPEx
- [ ] 3.3 `skills/insyra/SKILL.md` datafetch 段落加 TWStock 用法與「配合 `quant.Beta` 需先 Merge 對齊」一句
- [ ] 3.4 `CHANGELOG.md` 與 `CHANGELOG_TW.md` `## Unreleased` 新增 `### datafetch` 條目

## 4. Verification

- [ ] 4.1 `go test ./datafetch/...` 與 `go test ./...` 全綠且預設不觸網；`INSYRA_RUN_LIVE_TWSTOCK=1 go test ./datafetch/ -run TWStockLive` 在本機通過一次並把輸出貼進 PR；`golangci-lint run` 無新錯誤
- [ ] 4.2 `openspec validate add-datafetch-twstock --strict` 通過
