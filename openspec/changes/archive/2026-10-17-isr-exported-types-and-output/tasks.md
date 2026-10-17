## 1. Tests first

- [x] 1.1 Add `isr/output_test.go` covering exported types, `To` for CSV/JSON/Excel (file and memory), output options, refused targets and the `Row` key error; record that it fails to build against the old code (`To` undefined, `IncludeBOM` and `JSON.OutputOpts` missing)

## 2. Implementation

- [x] 2.1 Rename `dt`/`dl` to `DataTable`/`DataList`; keep `DT`, `DL` and the embedded field names
- [x] 2.2 Add `(*DataTable).To` and the `CSV`/`JSON`/`Excel` output options
- [x] 2.3 Name the isr type and offending key in `Row`/`Col` errors
- [x] 2.4 Doc comments: the exported types and the accepted inputs of `From`

## 3. Docs and records

- [x] 3.1 `Docs/isr.md`: the isr types, the output options, and a `To` section
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md` under `### isr`
- [x] 3.3 `api-review.md` I-2, I-3, I-5 marked fixed; `delivery-status.md` milestone
- [x] 3.4 Skills unchanged: no principle, workflow or documentation location changes

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`
- [x] 4.2 Push to `0.4`, CI green, archive, comment and close #237, #238, #239
