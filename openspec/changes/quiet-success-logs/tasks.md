# Tasks: quiet-success-logs

## 1. Tests first

- [x] 1.1 `success_logs_test.go` (package `insyra_test`) covers the save, convert and encoding scenarios, using a slog logger.
- [x] 1.2 A source-level test checks that `plot.SavePNG` and the three `py` functions do not call `insyra.LogInfo`.
- [x] 1.3 Record that both fail on the old code.

## 2. Implementation

- [x] 2.1 Delete the eleven Info lines in `csvxl`, `plot`, `gplot` and `py`.

## 3. Docs, rule, changelog

- [x] 3.1 `AGENTS.md` Key Conventions: what insyra logs at Info.
- [x] 3.2 `AGENTS.md` Follow-ups: encoding detection is unstable on a very short Big5 file.
- [x] 3.3 `Docs/Configuration.md`: what insyra logs at Info.
- [x] 3.4 `### Core` in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] 3.5 `api-review.md`: mark C-9 fixed. Add an entry to `delivery-status.md`.

## 4. Verification

- [x] 4.1 Run gofmt, `go build ./...`, `go vet ./...` and `go test ./...`.
- [x] 4.2 Run `golangci-lint run` and expect 0 issues.
- [x] 4.3 Run `openspec validate quiet-success-logs --strict`.
