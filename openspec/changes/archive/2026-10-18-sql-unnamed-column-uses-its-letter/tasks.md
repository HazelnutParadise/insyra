## 1. Tests first

- [x] 1.1 A table with unnamed columns is read back with their letters; a taken letter is refused and no table is created.

## 2. Implementation

- [x] 2.1 Name unnamed columns by letter before the write, on every path that collects the rows.

## 3. Docs and records

- [x] 3.1 `Docs/DataTable.md`.
- [x] 3.2 Entries in `CHANGELOG.md` and `CHANGELOG_TW.md`, marked BREAKING.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`.
