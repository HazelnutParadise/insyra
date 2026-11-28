# Proposal: sqlio-package (0.4)

## Why

Database reading and writing do not need to be `DataTable` methods, and they were the only reason the root package imported `database/sql`. #453 moves them to a package of their own, beside `parquet`, `arrowipc` and `excel`, with the call shape those packages already use. The design, confirmed by the owner on 2026-11-28, is in `/mnt/project-files/453/split-design.md`. `Query` and `StreamQuery` take `QueryOptions` rather than bare parameters so that a query keeps `ParseDates`, `DType`, `RowNameColumn` and batch reading, which `ReadSQLOptions.Query` had; that refinement was put to the owner on 2026-11-28.

Milestone: #453 root-package split, slice 3 of 3, `0.4` only. The branch stacks on `preprocess-package`; it waits for nothing else. `dev` gets no `sqlio`, because its SQL functions take `*gorm.DB` and a `0.4`-shaped package there would wrap gorm.

## What Changes

- **BREAKING** New package `sqlio`: `Read`, `Query`, `Stream`, `StreamQuery`, `Write`, `ReadOptions`, `QueryOptions`, `WriteOptions`, `Dialect` (`SQLite`, `MySQL`, `Postgres`) and `TableExistsPolicy` (`TableExistsFail`, `TableExistsReplace`, `TableExistsAppend`). The root's `ReadSQL`, `ReadSQLContext`, `ReadSQLStream`, `DataTable.ToSQL`, `DataTable.ToSQLContext`, `ReadSQLOptions`, `ToSQLOptions`, `SQLActionIfTableExists`, `SQLDialect` and their constants are removed, with `ToSQL` and `ToSQLContext` taken off `IDataTable`.
- `ReadSQLOptions.IndexCol`, a second name for `RowNameColumn`, is dropped. `ChunkSize` becomes a `batchSize` argument that must be positive.
- A result with no rows is an empty table instead of `nil`.
- The date layouts `ParseDates` uses move to `internal/prep`, shared by the core and `sqlio`.
- The CLI's `load sql` and `save … sql` call `sqlio` and behave as before.

## Capabilities

### New Capabilities

- `sqlio-package`: where database reading and writing live and what they take.

### Modified Capabilities

- `sql-database-connection`, `sql-identifier-quoting`, `sql-streaming-read`, `sql-table-replace`, `datalist-parse-dates`: the same behaviour, named by the `sqlio` functions.

## Impact

- Code: `sqlio/` (new), the removed `datatable_from_sql.go`, `datatable_to_sql.go`, `datatable_sql_conn.go`, `interfaces.go`, `enum_names.go`, `datalist_dates.go`, `internal/prep/`, `internal/dsl/commands/db_load.go`, `db_save.go`, `allpkgs/`, the root tests that read SQL.
- Docs: `Docs/sqlio.md` (new), `Docs/DataTable.md`, `Docs/DataList.md`, `Docs/Configuration.md`, `Docs/README.md`, `README.md`, `README_TW.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `AGENTS.md`.
