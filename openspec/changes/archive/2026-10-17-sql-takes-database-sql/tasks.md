## 1. Tests first
- [x] 1.1 Move the SQL tests from gorm to `*sql.DB`. Replace gorm's callback-based fault injection with a wrapping `database/sql` driver.
- [x] 1.2 Add tests for an unknown dialect, a custom query passed through as written, and the root package's dependency list. Record that they fail against the old code. Recorded 2026-10-17: against the gorm signatures the root test package does not compile (`cannot use db (variable of type *sql.DB) as *gorm.DB`, `undefined: SQLDialectSQLite`), and `go list -deps .` lists `gorm.io/gorm` and `github.com/Masterminds/squirrel`.

## 2. Implementation
- [x] 2.1 Add `SQLDialect` and change the five signatures in `datatable_to_sql.go`, `datatable_from_sql.go` and `interfaces.go`.
- [x] 2.2 Replace gorm's `Transaction`/`Exec`/`Raw` with `BeginTx`/`ExecContext`/`QueryContext`, and bind the internal statements per dialect.
- [x] 2.3 Build the batched INSERT without squirrel.
- [x] 2.4 Open the CLI `db` connections with the plain drivers, and make `DBConn.DB` a `*sql.DB`.
- [x] 2.5 Run `go mod tidy`.

## 3. Docs and records
- [x] 3.1 `Docs/DataTable.md` (and `Docs/cli-dsl.md` if it mentions gorm).
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`, marked BREAKING.
- [x] 3.3 E-3 in `api-review.md`, and an entry in `delivery-status.md`.

## 4. Verify
- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`.
