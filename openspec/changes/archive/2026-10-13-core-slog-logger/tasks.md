# Tasks: core-slog-logger

## 1. Tests first

- [x] 1.1 New tests in `logger_slog_test.go` for every scenario in `specs/core-log-output`. Record that they fail to compile against the old code, which has no `SetLogger`.

## 2. Implementation

- [x] 2.1 `config.go`: an atomic `*slog.Logger` field, `SetLogger`, `GetLogger`, and clearing it in `SetDefaultConfig`.
- [x] 2.2 `logger.go`: one output path the five `Log…` functions share. It sends a record with the right level, attributes and caller PC to the logger when one is set, and otherwise writes the old text.
- [x] 2.3 `error_buffer.go`: send the queue-full warning through that path.
- [x] 2.4 `internal/ccl`: a warning hook that defaults to the standard `log`. `init.go` points it at insyra's path, and the row-restore warning uses it.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/Configuration.md`: a section on sending logs to slog, covering what a record carries and how the level filters combine.
- [x] 3.2 Under `### Core` in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] 3.3 Mark K-17 fixed in `api-review.md`. Add an entry to `delivery-status.md`. Add an `AGENTS.md` follow-up for colour codes written when stderr is not a terminal.

## 4. Verification

- [x] 4.1 Run gofmt, `go build ./...`, `go vet ./...` and `go test ./...`, plus `-race` on the root package.
- [x] 4.2 Run `golangci-lint run` (0 issues) and `openspec validate core-slog-logger --strict`.
