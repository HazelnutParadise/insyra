# Tasks: sqlio-package (0.4)

## 1. Package

- [x] 1.1 Move the SQL reader, writer and dialect code into `sqlio/` with the new signatures; `Query` and `StreamQuery` with `QueryOptions`; `batchSize` must be positive; an empty result is an empty table.
- [x] 1.2 Remove the root names and the `IDataTable` methods; move the date layouts to `internal/prep`.
- [x] 1.3 The CLI's `load sql` and `save … sql` call `sqlio`; `allpkgs` registers it.

## 2. Tests

- [x] 2.1 Move the root SQL tests to `sqlio/`; the BLOB counting test builds its cells directly in the root and reads them from SQLite in `sqlio`.
- [x] 2.2 `sqlio/api_test.go`: two options values, table versus query, query settings, batch reading of a query, batch size below one, empty result, nil table, an embedding table, enum names.

## 3. Docs

- [x] 3.1 `Docs/sqlio.md`; pointers in `Docs/DataTable.md`; names in `Docs/DataList.md`, `Docs/Configuration.md`, `AGENTS.md`; rows in `README.md`, `README_TW.md`, `Docs/README.md`.
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`, BREAKING marked.

## 4. Verify

- [x] 4.1 `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate sqlio-package --strict`.
- [ ] 4.2 At archive, rewrite the Purpose of `sql-table-replace`, `sql-identifier-quoting` and `sql-streaming-read`, which name the old functions.
