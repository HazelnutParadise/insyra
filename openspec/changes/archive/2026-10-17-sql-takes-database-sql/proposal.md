# Proposal: sql-takes-database-sql

## Why

The root package imports `gorm.io/gorm` only because `ToSQL`, `ToSQLContext`, `ReadSQL`, `ReadSQLContext` and `ReadSQLStream` take a `*gorm.DB` (#235, E-3 in `api-review.md`). Every program that builds a `DataTable` therefore links gorm and its dependency graph. Measured on 2026-10-17, on darwin/arm64, with a program that only builds a `DataTable`:

- `go list -deps` lists 220 packages with the SQL code and 192 without it.
- Six third-party modules drop out: gorm, jinzhu/now, jinzhu/inflection, Masterminds/squirrel, lann/ps and lann/builder.
- The binary goes from 8.65 MB to 7.72 MB.

The code uses only these parts of gorm: `db.Name()` to tell the dialect, `Transaction`, `Exec`, `Raw().Rows()`, and gorm's rewrite of `?` to `$n` on PostgreSQL. Identifier quoting is insyra's own, and `Migrator` is never called. `database/sql` provides everything else, and gorm's `Transaction` is itself a wrapper around `BeginTx`, `Commit` and `Rollback`.

Other tools take the same approach. pandas' `read_sql` accepts a SQLAlchemy connectable or a DBAPI connection and keeps SQLAlchemy an optional dependency. R's DBI defines one interface and ships each driver separately. sqlx takes a `*sql.DB` together with a driver name and binds placeholders according to that name.

The owner ruled on 2026-10-17 (#235) for option A: `*sql.DB` plus an explicit dialect. Option B, a sub-package, was rejected. Under the one-name rule it would have kept gorm in the core for one more release behind Deprecated wrappers.

## What Changes

- **BREAKING**: `ToSQL(db *sql.DB, dialect SQLDialect, tableName, opts...)`, `ToSQLContext(ctx, db, dialect, tableName, opts...)`, `ReadSQL(db, dialect, tableName, opts...)`, `ReadSQLContext(ctx, db, dialect, tableName, opts...)` and `ReadSQLStream(ctx, db, dialect, tableName, opts...)`.
  - The names stay. Only the parameters change, so no name is removed and no Deprecated wrapper is needed.
  - gorm users pass `gormDB.DB()`.
- New type `SQLDialect`, with `SQLDialectSQLite`, `SQLDialectMySQL` and `SQLDialectPostgres`.
  - The dialect is a required parameter because it has no safe default.
  - An unknown dialect is an error before any statement runs. Before this change, an unknown dialect silently took the SQLite path.
- **BREAKING**: `ReadSQLOptions.Query` and `Params` reach the driver unchanged, so placeholders follow the driver's own syntax (`$1` on PostgreSQL).
  - gorm's rewrite of `?` to `$n`, its `@name` arguments and its `IN ?` slice expansion no longer apply.
  - insyra binds its own internal statements per dialect.
- The batched `INSERT` is built by insyra itself, so the squirrel dependency goes as well.
- **BREAKING (CLI Go API)**: `cli/commands.DBConn.DB` is a `*sql.DB`. The `db` commands open connections with the `database/sql` drivers the gorm drivers already wrapped: `github.com/glebarez/go-sqlite`, `github.com/go-sql-driver/mysql` and `github.com/jackc/pgx/v5/stdlib`.

## Capabilities

### New Capabilities

- `sql-database-connection`: what the SQL functions take as a connection and a dialect, and how parameters reach the driver.

### Modified Capabilities

- `sql-streaming-read`: the stream takes `*sql.DB` and a dialect. The nil-database scenario is unchanged.

## Impact

- `datatable_to_sql.go`, `datatable_from_sql.go`, `interfaces.go`, their tests, `internal/dsl/commands/db*.go`, `go.mod`.
- `Docs/DataTable.md`, `Docs/cli-dsl.md` (only if it shows gorm), both CHANGELOGs, `api-review.md`, `delivery-status.md`.
