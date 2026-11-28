// Package sqlio reads database tables and query results into DataTables and
// writes DataTables to database tables, through any database/sql connection.
//
//	dt, err := sqlio.Read(ctx, db, sqlio.Postgres, "orders", sqlio.ReadOptions{Limit: 100})
//	top, err := sqlio.Query(ctx, db, sqlio.Postgres, "SELECT * FROM orders WHERE total > $1",
//		sqlio.QueryOptions{Params: []any{500}})
//	err = sqlio.Write(ctx, dt, db, sqlio.Postgres, "orders_copy",
//		sqlio.WriteOptions{IfTableExists: sqlio.TableExistsReplace})
//
// Reading a table and running a query are two functions, as pandas keeps
// read_sql_table and read_sql_query apart: a table name is quoted and checked
// to exist, while a query goes to the driver as written. Stream and
// StreamQuery read the same two batch by batch.
//
// A *sql.DB does not say which database it talks to, so every function takes
// a Dialect, which decides how identifiers are quoted, how column types are
// inferred and how Replace swaps a table. Each function takes a context first
// and at most one options value.
//
// The package is separate from the core so that the core is the table and its
// operations; it imports only the standard library besides insyra.
package sqlio
