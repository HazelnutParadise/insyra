# Tasks

- [x] Classify HTTP 307 and 429 as `ErrTWStockThrottled` with an actionable message (`datafetch/twstock.go`).
- [x] Tests: 307 and 429 are throttling, 500 is not, two 307s then 200 succeeds with `Retries: 2`.
- [x] `Docs/datafetch.md`: "Being throttled" paragraph with starting values.
- [x] Entries under `## Unreleased` in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] `openspec validate twstock-report-throttling --strict` passes.
