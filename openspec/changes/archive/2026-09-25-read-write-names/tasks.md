# Tasks: read-write-names

## 1. Tests first

- [x] 1.1 `deprecated_names_test.go`: the notices and the old names' results

## 2. Implementation

- [x] 2.1 `ReadJSONFile`, `ToJSONBytes`, `ToJSONString`; the old names Deprecated wrappers; in-module callers moved; `IDataTable` gains the two methods

## 3. Docs and changelog

- [x] 3.1 `Docs/DataTable.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate read-write-names --strict`
