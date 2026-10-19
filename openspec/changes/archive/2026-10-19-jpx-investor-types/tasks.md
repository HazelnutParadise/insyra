# Tasks

- [x] `datafetch/internal/biff`: an OLE2 and BIFF8 reader for text, numbers, RK numbers and cached formula results, with a test that damaged files never panic.
- [x] `JPX`, `JPXClient.InvestorTypes`, `InvestorTypesContext`, `JPXUnit` (`datafetch/jpx.go`).
- [x] The weekly workbook and the older workbook parsers with their layout checks (`datafetch/jpx_parse.go`).
- [x] Fixtures from the live files; tests for both formats, the week-date rules around the year end, the full flow across both, shares against value, and bad calls.
- [x] Live checks across 2016, 2022, the 2025/2026 year end and 2026.
- [x] `Docs/datafetch.md` section, entries in both CHANGELOGs.
- [x] `openspec validate --specs --strict` passes.
