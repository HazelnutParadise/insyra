# Design: taifex-history

Measured on 2026-10-20 against `www.taifex.com.tw/cht/3/<table>Down` (GET works, so the requests are testable with the same fixtures as the OpenAPI):

| Table | Download | Reaches back to | Longest range |
| --- | --- | --- | --- |
| Institutional futures | `futContractsDateDown` | 2024 | at least 9 months |
| Institutional options | `callsAndPutsDateDown` | 2024 | at least 6 weeks |
| Large traders, futures | `largeTraderFutDown` | about 2004 | under 5 months |
| Large traders, options | `largeTraderOptDown` | about 2005 | not found |
| Put/call ratio | `pcRatioDown` | about 2003 | between 26 and 36 days |

Requests are made one calendar month at a time, which is inside every limit found. A date or range the site refuses comes back as an HTML page with `alert("日期時間錯誤 DateTime error")` instead of a CSV, and is returned as an error quoting that message and the range: silently returning an empty table would hide a request the exchange does not serve. The CSV headers are Chinese and Big5 encoded; the existing column definitions gained the website's headers as alternative keys, so the history tables have exactly the columns of the OpenAPI methods, and the site's `2026/10/12` dates are read next to the OpenAPI's `20261012`.

Names: `<Table>History(from, to)` beside the latest-day `<Table>()`, with `...Context` forms, like the rest of `TAIFEXClient`.
