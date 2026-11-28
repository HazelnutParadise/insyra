# Tasks: excel-package (0.3.x)

## 1. Package

- [x] 1.1 `excel/read.go` and `excel/write.go` on the core's functions; an empty `Sheet` reads the first sheet; more than one options value and a nil table are errors.
- [x] 1.2 Mark the core names and `csvxl.SheetNames` Deprecated, naming the replacement.
- [x] 1.3 `isr` and `cli/commands` write and list sheets through `excel`; `allpkgs` registers it.

## 2. Tests

- [x] 2.1 `excel/options_test.go`: the first sheet by default, a named sheet, `SheetNames`, two options values, a nil table, a round trip.

## 3. Docs

- [x] 3.1 `Docs/excel.md`; deprecation notes in `Docs/DataTable.md` and `Docs/csvxl.md`; rows in `README.md`, `README_TW.md`, `Docs/README.md`.
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`.

## 4. Verify

- [x] 4.1 `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate excel-package --strict`.
