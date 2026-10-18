# Tasks: quiet-success-logs

## 1. Tests first

- [x] 1.1 `success_logs_test.go`: a recording logger sees nothing from the chart savers and only encoding lines from the converters; the source of `SavePNG` and the `py` functions has no `LogInfo`; a guessed encoding is still logged

## 2. Implementation

- [x] 2.1 delete the success lines in `csvxl`, `plot`, `gplot` and `py`

## 3. Docs and changelog

- [x] 3.1 `AGENTS.md` convention and follow-up, `Docs/Configuration.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate quiet-success-logs --strict`
