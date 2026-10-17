# Tasks: exported-functions-are-functions

## 1. Tests first

- [x] 1.1 `deprecated_names_test.go`: the wrapper's notice and its result

## 2. Implementation

- [x] 2.1 `ReadSlice2D` as a function, `Slice2DToDataTable` as a Deprecated wrapper; in-module callers moved

## 3. Docs and changelog

- [x] 3.1 `Docs/DataTable.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate exported-functions-are-functions --strict`
