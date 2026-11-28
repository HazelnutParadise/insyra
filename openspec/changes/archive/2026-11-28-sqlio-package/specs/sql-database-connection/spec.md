## MODIFIED Requirements

### Requirement: The SQL functions take database/sql and an explicit dialect

`sqlio.Write`, `sqlio.Read`, `sqlio.Query`, `sqlio.Stream` and `sqlio.StreamQuery` SHALL take a `context.Context`, a `*sql.DB` and a `sqlio.Dialect`. The root package SHALL NOT import `database/sql`, `gorm.io/gorm` or `github.com/Masterminds/squirrel`. A dialect other than `sqlio.SQLite`, `sqlio.MySQL` or `sqlio.Postgres` SHALL be an error before any statement runs.

#### Scenario: Round trip through database/sql
- **WHEN** a table is written with `sqlio.Write(ctx, dt, db, sqlio.SQLite, "t")` to a SQLite `*sql.DB` and read back with `sqlio.Read(ctx, db, sqlio.SQLite, "t")`
- **THEN** the values come back unchanged

#### Scenario: Unknown dialect
- **WHEN** `sqlio.Write` or `sqlio.Read` is called with `sqlio.Dialect("oracle")`
- **THEN** it returns an error naming the dialect, and the database is left as it was

#### Scenario: The core does not link gorm
- **WHEN** `go list -deps` runs on the root package
- **THEN** none of `database/sql`, `gorm.io/gorm` and `github.com/Masterminds/squirrel` appears

### Requirement: A custom query reaches the driver unchanged

The query given to `sqlio.Query` or `sqlio.StreamQuery`, and its `QueryOptions.Params`, SHALL be passed to the driver as given, with no placeholder rewriting.

#### Scenario: Driver placeholders
- **WHEN** `sqlio.Query` runs `SELECT * FROM t WHERE id = ?` with `Params` `{2}` against SQLite
- **THEN** only the row with id 2 is returned

### Requirement: An unnamed column is written under its letter

`sqlio.Write` SHALL write a column that has no name under its Excel-style letter, counted among the table's columns. When another column already has that letter as its name, compared without case, `sqlio.Write` SHALL return an error before creating or changing anything.

#### Scenario: Unnamed columns
- **WHEN** a table whose columns are unnamed, `label` and unnamed is written and read back
- **THEN** the columns are named `A`, `label` and `C`

#### Scenario: Letter taken
- **WHEN** the first column is unnamed and the second is named `a`
- **THEN** `sqlio.Write` returns an error naming column `A`, and no table is created

#### Scenario: Servers accept it
- **WHEN** a table with one unnamed column is written to MySQL or PostgreSQL
- **THEN** the write succeeds, and the column reads back as `A`
