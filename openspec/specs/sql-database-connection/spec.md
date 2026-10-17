# sql-database-connection Specification

## Purpose
SQL 讀寫只依賴 Go 標準函式庫的 `database/sql`：呼叫端傳入 `*sql.DB` 並明講資料庫種類，核心套件因此不必連帶拉進 gorm。自訂查詢原封不動交給驅動程式，寫入失敗時資料表保持原樣的保證和改用前相同。
## Requirements
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

### Requirement: An unnamed column is written under its letter

`ToSQL` SHALL write a column that has no name under its Excel-style letter, counted among the table's columns. When another column already has that letter as its name, compared without case, `ToSQL` SHALL return an error before creating or changing anything.

#### Scenario: Unnamed columns
- **WHEN** a table whose columns are unnamed, `label` and unnamed is written and read back
- **THEN** the columns are named `A`, `label` and `C`

#### Scenario: Letter taken
- **WHEN** the first column is unnamed and the second is named `a`
- **THEN** `ToSQL` returns an error naming column `A`, and no table is created

#### Scenario: Servers accept it
- **WHEN** a table with one unnamed column is written to MySQL or PostgreSQL
- **THEN** the write succeeds, and the column reads back as `A`

