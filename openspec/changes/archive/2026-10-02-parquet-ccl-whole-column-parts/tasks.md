# Tasks: parquet-ccl-whole-column-parts

## 1. internal/ccl

- [x] 1.1 `StreamRefusal`: the reason `ResolveWholeTable` would refuse an expression, nil when it would not; `ReferencedColumns`: the column positions a bound expression reads, every column for `@` (failing tests first)

## 2. parquet

- [x] 2.1 `FilterWithCCL` and `ApplyCCL` take the whole-column path for an expression or statement the streaming path cannot compute, reading only the columns it reads and evaluating it the way the `DataTable` CCL methods do; results and errors compared with the loaded table (failing tests first)

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/parquet.md`: which parts hold whole columns, what they hold and the extra read
- [x] 3.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `parquet`
- [x] 3.3 `AGENTS.md`: resolve the follow-up on refused parts; `delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate parquet-ccl-whole-column-parts --strict`
