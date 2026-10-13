# Tasks: color-only-on-terminals

## 1. Tests first

- [x] 1.1 Write `color_terminal_test.go`: one test per scenario in `specs/colored-output`. Use a substitute for `isTerminal` to test the terminal path.
- [x] 1.2 Record the failures on the old code. Both non-terminal tests fail with colour codes in the buffer.

## 2. Implementation

- [x] 2.1 `utils.go`: `colorText(w, code, text)` colours only for a terminal destination. Add the replaceable `isTerminal`.
- [x] 2.2 `logger.go`: check `log.Writer()` and put the reset before the newline.
- [x] 2.3 Pass the destination writer to `colorText` in every caller: `show.go`, `show_head_tail.go`, `datalist_summary.go` and `datatable_summary.go`.

## 3. Docs, changelog

- [x] 3.1 Document when output is coloured in `Docs/Configuration.md`, and add it to the `ShowTo` and `SummaryTo` notes in `Docs/DataList.md` and `Docs/DataTable.md`.
- [x] 3.2 Add the change under `### Core` in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] 3.3 Remove the follow-up from `AGENTS.md` and add an entry to `delivery-status.md`.

## 4. Verification

- [x] 4.1 Run gofmt, `go build ./...`, `go vet ./...` and `go test ./...`, with `-race` on the root package.
- [x] 4.2 Run `golangci-lint run` and expect 0 issues.
- [x] 4.3 Run `openspec validate color-only-on-terminals --strict`.
