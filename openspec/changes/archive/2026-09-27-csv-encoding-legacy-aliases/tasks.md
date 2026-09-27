# Tasks

## 1. Aliases in the decoder table

- [x] 1.1 Add a test that reads through `DecodingReader` and `csvxl.ReadCsvToString` with each alias (both cases), a `utf-8-sig` file carrying a byte-order mark, and a name no charset owns; it fails before the table change (`go test ./internal/csv/ ./csvxl/`)
- [x] 1.2 Add the aliases to `decoders` in `internal/csv/decoder.go`, `utf8sig` and `utf8bom` as `unicode.UTF8BOM`; the test in 1.1 passes
- [x] 1.3 `Docs/csvxl.md` names the aliases; both changelogs gain an entry under `## Unreleased` → `` ### `csvxl` `` or the section holding the refusal entry

## 2. Verification

- [x] 2.1 `go test ./...`, `golangci-lint run` and `openspec validate csv-encoding-legacy-aliases --strict` pass
