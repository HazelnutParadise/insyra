# Tasks: stats-result-strings

## 1. Tests first

- [x] 1.1 renderer rules, one output test per family of result, and the test that every result type has both methods

## 2. Implementation

- [x] 2.1 `stats/result_text.go` renderer, `stats/result_strings.go` methods
- [x] 2.2 `ChiSquareTestResult.Show` and `FactorAnalysisResult.Show` use it

## 3. Docs and changelog

- [x] 3.1 `Docs/stats.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: `stats`

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate stats-result-strings --strict`
