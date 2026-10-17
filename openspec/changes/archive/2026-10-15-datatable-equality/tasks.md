# Tasks: datatable-equality

## 1. Tests first

- [x] 1.1 `datatable_equality_test.go`: names that differ with equal cells, a cell of another type, NaN, a different shape, a short column read as nil, a clone, one row name, one column name, the table name, the same table, a nil argument

## 2. Implementation

- [x] 2.1 `datatable_equality.go`: `IsEqualTo` and `IsTheSameAs`, locking both tables with `AtomicDoAll`
- [x] 2.2 `interfaces.go`: `IDataTable` gains both methods

## 3. Docs and changelog

- [x] 3.1 `Docs/DataTable.md`: a Comparison section
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate datatable-equality --strict`
