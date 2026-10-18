# Tasks: py-typed-run

## 1. Tests first

- [x] 1.1 `py/run_test.go`: typed decode, DataFrame, zero value on error, table by value refused, cancel, nil and ended contexts, timeout meaning kept

## 2. Implementation

- [x] 2.1 `Run[T]`, the pip context forms, the nil and ended context checks, the deprecation

## 3. Docs and changelog

- [x] 3.1 `Docs/py.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: `py`
- [x] 3.3 `AGENTS.md` follow-up to remove `RunCodeWithTimeout` in 0.4

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate py-typed-run --strict`
