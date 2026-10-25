# Tasks

- [x] Tests first: `read_encoding_name_test.go` (unknown names refused by ReadCSV, ReadCSV_FileWithOptions and StreamCSV; known names and cp950 still decode) and `csvxl/encoding_names_test.go` (CsvToExcel refuses and writes nothing).
- [x] Check an explicit `Encoding` with `CheckDecodable` in `read.go`; check it up front in `CsvToExcel` and `AppendCsvToExcel`.
- [x] Add `cp950` and `cp936` to the decoder table.
- [x] `Docs/DataTable.md` and the `CSVReadOptions` comment.
- [x] Entries under `## Unreleased` / `### Core` in both CHANGELOGs.
- [x] `openspec validate csv-refuse-unknown-encoding --strict` passes.
