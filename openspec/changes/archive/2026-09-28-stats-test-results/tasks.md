# Tasks: stats-test-results

## 1. Implementation

- [x] 1.1 export `TestResult`, add `HypothesisTestResult` and `Base`
- [x] 1.2 test that every result type satisfies the interface (in `stats/result_strings_test.go`)

## 2. Docs and changelog

- [x] 2.1 `Docs/stats.md`
- [x] 2.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: `stats`

## 3. Verification

- [x] 3.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate stats-test-results --strict`
