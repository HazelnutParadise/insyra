# sqlio-package Specification

## Purpose
Keeps database reading and writing in their own `sqlio` package, so the core package does not depend on `database/sql`: `Read` and `Stream` take a table, `Query` and `StreamQuery` take SQL, and `Write` saves any table, each with one optional options value.

## Requirements

### Requirement: Database reading and writing live in the sqlio package

Reading a database table or a query result into a DataTable and writing a DataTable to a table SHALL be provided by the package `sqlio`, and the root package `insyra` SHALL NOT declare `ReadSQL`, `ReadSQLContext`, `ReadSQLStream`, `DataTable.ToSQL`, `DataTable.ToSQLContext` or their option and constant types. Each `sqlio` function SHALL take at most one options value and SHALL return an error naming the options type when given more.

#### Scenario: Reading a table
- **WHEN** `sqlio.Read(ctx, db, sqlio.SQLite, "t", sqlio.ReadOptions{Columns: []string{"a"}})` reads a table with columns `a` and `b`
- **THEN** it returns a table holding only `a`

#### Scenario: Two options values
- **WHEN** `sqlio.Write(ctx, dt, db, sqlio.SQLite, "t", sqlio.WriteOptions{}, sqlio.WriteOptions{})` is called
- **THEN** it returns an error that names `WriteOptions` and writes nothing

### Requirement: A table and a query are read by separate functions

`sqlio.Read` and `sqlio.Stream` SHALL take a table name, quote it and check that the table exists, and SHALL refuse an empty name with an error that names `Query` or `StreamQuery`. `sqlio.Query` and `sqlio.StreamQuery` SHALL take a query and `QueryOptions` holding `Params`, `RowNameColumn`, `ParseDates` and `DType`, applied as `Read` applies them, and SHALL refuse an empty query.

#### Scenario: Text after a table name
- **WHEN** `sqlio.Read` is given the table name `t WHERE 1=1`
- **THEN** it returns an error saying the table does not exist

#### Scenario: Settings on a query
- **WHEN** `sqlio.Query` runs a query with `ParseDates: []string{"at"}` and `DType` forcing `n` to float64
- **THEN** `at` holds `time.Time` values and `n` holds float64 values

### Requirement: A result with no rows is an empty table

`sqlio.Read` and `sqlio.Query` SHALL return a non-nil DataTable with no rows and a nil error when the result has no rows.

#### Scenario: No matching rows
- **WHEN** `sqlio.Query` runs a query that matches no row
- **THEN** it returns a table with 0 rows and a nil error

### Requirement: Write takes any table

`sqlio.Write` SHALL take an `insyra.IDataTable`, read it in one consistent snapshot, and SHALL return an error for a nil table, including a nil pointer behind the interface, before running any statement.

#### Scenario: A nil table
- **WHEN** `sqlio.Write(ctx, (*insyra.DataTable)(nil), db, sqlio.SQLite, "t")` is called
- **THEN** it returns an error and creates no table

### Requirement: TableExistsPolicy prints its name

`sqlio.TableExistsPolicy` SHALL print the name of each constant and `TableExistsPolicy(n)` for any other value.

#### Scenario: Printing
- **WHEN** `fmt.Sprint(sqlio.TableExistsReplace)` and `fmt.Sprint(sqlio.TableExistsPolicy(99))` run
- **THEN** they give `TableExistsReplace` and `TableExistsPolicy(99)`
