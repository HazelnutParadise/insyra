## ADDED Requirements

### Requirement: The SQL functions take database/sql and an explicit dialect

`ToSQL`, `ToSQLContext`, `ReadSQL`, `ReadSQLContext` and `ReadSQLStream` SHALL take a `*sql.DB` and an `insyra.SQLDialect`. The root package SHALL NOT import `gorm.io/gorm` or `github.com/Masterminds/squirrel`. A dialect other than `SQLDialectSQLite`, `SQLDialectMySQL` or `SQLDialectPostgres` SHALL be an error before any statement runs.

#### Scenario: Round trip through database/sql
- **WHEN** a table is written with `ToSQL(db, SQLDialectSQLite, "t")` to a SQLite `*sql.DB` and read back with `ReadSQL(db, SQLDialectSQLite, "t")`
- **THEN** the values come back unchanged

#### Scenario: Unknown dialect
- **WHEN** `ToSQL` or `ReadSQL` is called with `SQLDialect("oracle")`
- **THEN** it returns an error naming the dialect, and the database is left as it was

#### Scenario: The core does not link gorm
- **WHEN** `go list -deps` runs on the root package
- **THEN** neither `gorm.io/gorm` nor `github.com/Masterminds/squirrel` appears

### Requirement: A custom query reaches the driver unchanged

`ReadSQLOptions.Query` and `ReadSQLOptions.Params` SHALL be passed to the driver as given, with no placeholder rewriting.

#### Scenario: Driver placeholders
- **WHEN** `Query` is `SELECT * FROM t WHERE id = ?` with `Params` `{2}` against SQLite
- **THEN** only the row with id 2 is returned

### Requirement: A failed write still leaves the table as it was

Replacing or appending through `*sql.DB` SHALL keep the transactional guarantees the gorm version had. A failed write SHALL leave the existing table and its rows unchanged on SQLite and PostgreSQL, and through the staging-table rename on MySQL.

#### Scenario: Insert fails during Replace
- **WHEN** Replace runs on an existing table and one of the INSERTs fails
- **THEN** the call returns the error, and the table holds its old rows
