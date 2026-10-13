# Tasks: encoding-guess-refuses-ties

## 1. Tests first

- [x] 1.1 `encoding_guess_test.go`: a tie, a header over numbers, a weak guess, a clear guess, and a reader input. On the old code with a stub sentinel, the four refusal tests fail and the clear-guess test passes.
- [x] 1.2 `csvxl/read_csv_encoding_test.go`: `ReadCSVToString` and `CSVToExcel` refuse the ambiguous file, and read it once `Big5` is named.
- [x] 1.3 `internal/dsl/commands/load_save_test.go`: the CLI error carries the hint, and `encoding big5` reads the file.
- [x] 1.4 `batch6_test.go`: the fallback test uses `0xFF`, which no recogniser names. `0xE9` is named UTF-8 at 10 and is now refused.

## 2. Implementation

- [x] 2.1 `utils.go`:
  - add `ErrEncodingAmbiguous` and `minEncodingConfidence`;
  - add `conclusiveCharset`, which sorts stably, refuses a tie or a score below 50, and lists the candidates;
  - switch to `DetectAll`.
- [x] 2.2 `internal/dsl/commands/load.go`: add the hint when the CSV read fails with `ErrEncodingAmbiguous`.

## 3. Docs, changelog

- [x] 3.1 Document the rule, the error and the Shift_JIS weak spot in `Docs/utils.md`, `Docs/csvxl.md`, `Docs/DataTable.md` and `Docs/cli-dsl.md`.
- [x] 3.2 Add a **BREAKING** entry under `### Core` in `CHANGELOG.md` and `CHANGELOG_TW.md`.
- [x] 3.3 `AGENTS.md`: replace the follow-up with one for the Shift_JIS read that is still wrong.
- [x] 3.4 Add an entry to `delivery-status.md`.

## 4. Verification

- [x] 4.1 Run gofmt, `go build ./...`, `go vet ./...` and `go test ./...`.
- [x] 4.2 Run `golangci-lint run` and expect 0 issues.
- [x] 4.3 Run `openspec validate encoding-guess-refuses-ties --strict`.
