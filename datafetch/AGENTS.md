# AGENTS.md (datafetch/)

Rules for working in `datafetch/`. The top-level [AGENTS.md](/AGENTS.md) covers the wider repo.

## Known pitfalls

- TAIFEX's `futContractsDateDown` and `callsAndPutsDateDown` serve a rolling window of the last three years (the site's note: 自2012年5月1日起提供交易日前三年之資料), so describe that limit as rolling, never as a fixed first year. For older institutional data, FinMind's `TaiwanFuturesInstitutionalInvestors` and `TaiwanOptionInstitutionalInvestors` match TAIFEX cell for cell; FinMind orders the groups 投信, 外資, 自營商 and says 外資, not 外資及陸資.
- A TPEx `/www/zh-tw/...` JSON endpoint with server paging returns only 10 rows of `tables[0].data` and no error; send `paging-size=10000` and compare `len(data)` with `totalCount`, since `page`, `limit`, `length`, `pageSize` and POST do nothing.
- When changing TWStock parsing, also run `INSYRA_RUN_LIVE_TWSTOCK=1 go test -run TestTWStockLive ./datafetch/` against an old month: the fixtures hold recent months only, and older months use other headers (TPEx heads volume 成交仟股 before about 2025).
- Listing and delisting sources: TWSE OpenAPI `/v1/opendata/t187ap03_L` (上市日期), TPEx `/openapi/v1/mopsfin_t187ap03_O` (`DateOfListing`), TWSE `/rwd/zh/company/suspendListing?response=json` (key `status`, not `stat`; 2001 on, one reply, no reason) and TPEx `/www/zh-tw/company/deListed?date=<year>&reason=-1|2&paging-size=10000&response=json` (reason 2, moved to TWSE, is a subset of -1). Stock codes are reused, so key a listing on code and dates, not code alone.
