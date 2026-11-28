# [ sqlio ] Package

The `sqlio` package reads a database table or the result of a query into a `DataTable`, and writes a `DataTable` to a database table, through any `database/sql` connection.

Reading a table and running a query are two functions, the way pandas keeps `read_sql_table` and `read_sql_query` apart: a table name is quoted and checked to exist, while a query goes to the driver as written. `Stream` and `StreamQuery` read the same two a batch at a time. The package is separate from the core so that the core is the table and its operations.

```go
import "github.com/HazelnutParadise/insyra/sqlio"
```

Every function takes a context first, then the connection and its dialect, and at most one options value; giving more than one is an error, never "first wins".

## Table of Contents

- [Data Structures](#data-structures)
  - [Dialect](#dialect)
  - [ReadOptions](#readoptions)
  - [QueryOptions](#queryoptions)
  - [WriteOptions](#writeoptions)
  - [TableExistsPolicy](#tableexistspolicy)
- [Main Functions](#main-functions)
  - [Read](#read)
  - [Query](#query)
  - [Stream and StreamQuery](#stream-and-streamquery)
  - [Write](#write)
- [Moved from the core package](#moved-from-the-core-package)

## Data Structures

### Dialect

```go
type Dialect string

const (
    SQLite   Dialect = "sqlite"
    MySQL    Dialect = "mysql"
    Postgres Dialect = "postgres"
)
```

A `*sql.DB` does not say which database it talks to, so every function takes the dialect and it has no default; any other value is an error. It decides how names are quoted, which column types are inferred, how insyra's own statements bind their values and how a table is replaced. Name a MySQL-compatible server as `MySQL`.

### ReadOptions

```go
type ReadOptions struct {
    RowNameColumn string                  // column whose values become row names; "" means "row_name"
    Columns       []string                // SELECT only these columns, in this order
    Schema        string                  // schema (PostgreSQL) or database (MySQL); SQLite ignores it
    Limit         int                     // at most this many rows; 0 means no limit
    Offset        int                     // skip this many rows first
    WhereClause   string                  // body of a WHERE clause, without the keyword
    OrderBy       string                  // body of an ORDER BY clause, without the keywords
    Params        []any                   // values for the placeholders written in WhereClause
    ParseDates    []string                // columns whose text becomes time.Time
    DType         map[string]reflect.Type // force the Go type of these columns
}
```

The zero value reads every column and row, and takes a column named `row_name`, when there is one, as the row names; that column is then not a column of the result. `RowNameColumn` and `ParseDates` match column names without regard to case.

`WhereClause` and `OrderBy` are written into the SQL as they are, so never place untrusted input in them: write placeholders and bind the values through `Params`, in the driver's own form (`?` on SQLite and MySQL, `$1` on PostgreSQL), or use `Query`.

`ParseDates` tries RFC 3339 (with and without fractions), `2006-01-02 15:04:05`, `2006-01-02T15:04:05` and `2006-01-02`, and gives the instant in UTC. `DType` recognises `reflect.TypeFor` of the integer and float types, `bool`, `string`, `time.Time` and `[]byte`; a column `DType` forces is left out of `ParseDates`.

### QueryOptions

```go
type QueryOptions struct {
    Params        []any                   // values for the query's placeholders
    RowNameColumn string                  // as in ReadOptions
    ParseDates    []string                // as in ReadOptions
    DType         map[string]reflect.Type // as in ReadOptions
}
```

A query says for itself which columns, rows and order it wants, so `QueryOptions` holds only the parameters and how the result is turned into a table.

### WriteOptions

```go
type WriteOptions struct {
    IfTableExists TableExistsPolicy // TableExistsFail (default), TableExistsReplace or TableExistsAppend
    HasRowNames   bool              // write the row names as a first column named row_name
    ColumnTypes   map[string]string // SQL type of the named columns, such as "VARCHAR(64)"
    Schema        string            // schema (PostgreSQL) or database (MySQL); SQLite ignores it
    BatchSize     int               // rows per INSERT; 0 means 500
}
```

The zero value creates the table, refuses one that is already there, writes no row names, infers each column's type and inserts 500 rows per statement. `BatchSize` times the number of columns must stay below the driver's limit on bound values (65535 on PostgreSQL and MySQL). `Schema` is written as given, so quote it yourself if it needs quoting.

### TableExistsPolicy

```go
type TableExistsPolicy int

const (
    TableExistsFail    TableExistsPolicy = iota // refuse and leave the table as it was (the zero value)
    TableExistsReplace                          // replace the table with one holding only the new rows
    TableExistsAppend                           // keep the table, add missing columns, append the rows
)
```

`TableExistsPolicy` prints its constant's name, and an unknown value as `TableExistsPolicy(n)`. The names match `excel.SheetExistsPolicy`.

## Main Functions

### Read

```go
func Read(ctx context.Context, db *sql.DB, dialect Dialect, table string, opts ...ReadOptions) (*insyra.DataTable, error)
```

**Description:** Reads a table into a new `DataTable`. `db` is a connection opened with any driver; with gorm, pass the one `gormDB.DB()` returns. The table name is quoted for the dialect and checked to exist first, so a missing table is an error naming it and text such as `t WHERE 1=1` is looked up as a name, never run as SQL. An empty name is an error that points to `Query`. The query and the scan run under `ctx`.

**Type handling:** When the driver returns numeric or boolean columns as `[]byte` (common with the MySQL driver), the column's database type decides the Go type. Binary columns (`BLOB`, `BYTEA`, `BINARY`, `VARBINARY`) stay `[]byte`. `DType` takes precedence over both. A result with no rows is an empty table, not `nil`.

**Example:**

```go
// Any database/sql driver works; this one is github.com/go-sql-driver/mysql.
db, err := sql.Open("mysql", "user:pass@tcp(localhost:3306)/shop")
if err != nil {
    log.Fatal(err)
}
dt, err := sqlio.Read(ctx, db, sqlio.MySQL, "users", sqlio.ReadOptions{
    Limit:   100,
    OrderBy: "id DESC",
})

// With a WHERE clause whose value is bound, not written into the SQL.
active, err := sqlio.Read(ctx, db, sqlio.MySQL, "users", sqlio.ReadOptions{
    WhereClause: "age > ?",
    Params:      []any{18},
})

// Already using gorm: pass the connection underneath it.
sqlDB, err := gormDB.DB()
dt, err = sqlio.Read(ctx, sqlDB, sqlio.Postgres, "users")
```

### Query

```go
func Query(ctx context.Context, db *sql.DB, dialect Dialect, query string, opts ...QueryOptions) (*insyra.DataTable, error)
```

**Description:** Runs a query of your own and reads its result into a new `DataTable`. The query and its `Params` go to the driver as written, so they use the driver's own placeholders. An empty query is an error.

**Example:**

```go
top, err := sqlio.Query(ctx, db, sqlio.Postgres,
    "SELECT id, total, placed_at FROM orders WHERE total > $1 ORDER BY total DESC",
    sqlio.QueryOptions{
        Params:     []any{500},
        ParseDates: []string{"placed_at"},
        DType:      map[string]reflect.Type{"total": reflect.TypeFor[float64]()},
    })
```

### Stream and StreamQuery

```go
func Stream(ctx context.Context, db *sql.DB, dialect Dialect, table string, batchSize int, opts ...ReadOptions) iter.Seq2[*insyra.DataTable, error]
func StreamQuery(ctx context.Context, db *sql.DB, dialect Dialect, query string, batchSize int, opts ...QueryOptions) iter.Seq2[*insyra.DataTable, error]
```

**Description:** Read a table, or the result of a query, `batchSize` rows at a time, for results that do not fit in memory. Each batch arrives as a `DataTable` with a nil error. A failure (building or running the query, scanning a row, `ctx` being cancelled) arrives once, as a nil table with the error, and ends the loop. The query runs when the loop starts and batches are read in your own goroutine, so leaving the loop early closes the rows and returns the connection to the pool at once, with nothing left running. `batchSize` must be positive; a smaller value is an error.

**Example:**

```go
for batch, err := range sqlio.Stream(ctx, db, sqlio.MySQL, "events", 5000) {
    if err != nil {
        log.Fatal(err)
    }
    process(batch)
}

for batch, err := range sqlio.StreamQuery(ctx, db, sqlio.Postgres,
    "SELECT * FROM events WHERE kind = $1", 5000, sqlio.QueryOptions{Params: []any{"click"}}) {
    if err != nil {
        log.Fatal(err)
    }
    process(batch)
}
```

### Write

```go
func Write(ctx context.Context, dt insyra.IDataTable, db *sql.DB, dialect Dialect, table string, opts ...WriteOptions) error
```

**Description:** Writes the table to a database table. Every statement runs under `ctx`. A nil table, including a nil pointer behind the interface, and a table with no rows are errors.

**Behavior:**

- On SQLite and PostgreSQL every statement (CREATE TABLE, DROP TABLE, ALTER TABLE, INSERT) runs in one transaction. If any step fails, the transaction is rolled back and the database is left as it was.
- MySQL commits every CREATE, ALTER, DROP and RENAME statement on its own, ending any transaction it is in (MySQL Reference Manual, [Statements That Cause an Implicit Commit](https://dev.mysql.com/doc/refman/8.4/en/implicit-commit.html)), so a transaction cannot undo them there:
  - `TableExistsReplace` writes the rows into a new staging table named `insyra_new_<random>`, then puts it in place with one `RENAME TABLE` statement, which MySQL runs atomically, and drops the old table. A write that fails, or is cancelled, drops the staging table and leaves the old table as it was. The staging table and the renamed old table have names of their own, so the account needs the CREATE, INSERT, ALTER and DROP privileges on the database, not only on the table being replaced; without them the call fails before anything changes. A table that another table references by foreign key is refused, also before anything changes, because the rename would move that reference to the old copy. If the old copy cannot be dropped after the swap, the call returns an error naming `insyra_old_<random>`, which still holds the old rows, while the target already holds the new ones. When the table does not exist, Replace creates it; if another session creates it meanwhile, the call fails with `already exists` instead of dropping that table.
  - A missing table is written under a staging name `insyra_new_<random>` in one transaction and renamed in once every row is written, so a write that fails leaves no table behind. If another session creates the table meanwhile, the rename fails, that table is kept, and the call fails with `already exists`. This needs the same CREATE, INSERT, ALTER and DROP privileges on the database as Replace.
  - With `TableExistsAppend` the missing columns are added first, which commits them, and the rows are then written in one transaction. If the rows fail, the added columns are dropped again; a column that cannot be dropped is named in the error.
- Rows are inserted with batched multi-row INSERT statements.
- With `TableExistsAppend`, the existing table's columns are fetched once and any missing ones are added with `ALTER TABLE` before inserting; each added column is logged at Info.
- Type inference looks at the first non-nil value in each column. Recognised types include `time.Time` (`DATETIME` or `TIMESTAMP`), `[]byte` (`BLOB` or `BYTEA`), the `sql.Null*` types (their value) and pointers (what they point to). A column of only nils is `TEXT`. `ColumnTypes` overrides the inferred type.
- A column with no name is written under its letter (`A`, `B`, …), because MySQL and PostgreSQL refuse an empty column name. If another column is already named that letter, compared without case, the call fails before anything is written.

**Example:**

```go
err := sqlio.Write(ctx, dt, db, sqlio.MySQL, "users",
    sqlio.WriteOptions{IfTableExists: sqlio.TableExistsReplace})
if err != nil {
    log.Fatal(err)
}

// Append into an existing table; missing columns are added once.
err = sqlio.Write(ctx, dt, db, sqlio.MySQL, "events", sqlio.WriteOptions{
    IfTableExists: sqlio.TableExistsAppend,
    BatchSize:     1000,
})
```

## Moved from the core package

These names used to live in the root `insyra` package and no longer exist there:

| Before | Now |
|---|---|
| `insyra.ReadSQL(db, dialect, table, opts)` | `sqlio.Read(context.Background(), db, dialect, table, opts)` |
| `insyra.ReadSQLContext(ctx, db, dialect, table, opts)` | `sqlio.Read(ctx, db, dialect, table, opts)` |
| `insyra.ReadSQL(db, dialect, "", insyra.ReadSQLOptions{Query: q, Params: p})` | `sqlio.Query(ctx, db, dialect, q, sqlio.QueryOptions{Params: p})` |
| `insyra.ReadSQLStream(ctx, db, dialect, table, insyra.ReadSQLOptions{ChunkSize: n})` | `sqlio.Stream(ctx, db, dialect, table, n)`; with `Query` set, `sqlio.StreamQuery(ctx, db, dialect, q, n)` |
| `dt.ToSQL(db, dialect, table, opts)`, `dt.ToSQLContext(ctx, db, dialect, table, opts)` | `sqlio.Write(ctx, dt, db, dialect, table, opts)` |
| `insyra.ReadSQLOptions` | `sqlio.ReadOptions`, or `sqlio.QueryOptions` for a query |
| `ReadSQLOptions.IndexCol` | `RowNameColumn`, its only name now |
| `ReadSQLOptions.ChunkSize` (0 meant 1000) | the `batchSize` argument, which must be positive |
| `insyra.ToSQLOptions`, its `IfExists` field | `sqlio.WriteOptions`, its `IfTableExists` field |
| `insyra.SQLActionIfTableExists`, `SQLActionIfTableExistsFail`, `Replace`, `Append` | `sqlio.TableExistsPolicy`, `TableExistsFail`, `TableExistsReplace`, `TableExistsAppend` |
| `insyra.SQLDialect`, `SQLDialectSQLite`, `SQLDialectMySQL`, `SQLDialectPostgres` | `sqlio.Dialect`, `sqlio.SQLite`, `sqlio.MySQL`, `sqlio.Postgres` |

Reading and writing behave as before, except that a result with no rows is now an empty table where `ReadSQL` returned `nil`. The CLI commands `load sql` and `save … sql` are unchanged.
