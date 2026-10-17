package insyra

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// SQLDialect names the database behind a *sql.DB passed to ToSQL, ReadSQL and
// ReadSQLStream. It decides how identifiers are quoted, how column types are
// inferred, how insyra's own statements bind their values, and how Replace
// swaps a table. A *sql.DB does not say which database it talks to, so the
// caller names it; there is no default.
type SQLDialect string

const (
	// SQLDialectSQLite is SQLite, through any database/sql driver for it.
	SQLDialectSQLite SQLDialect = "sqlite"
	// SQLDialectMySQL is MySQL, and servers that speak its dialect.
	SQLDialectMySQL SQLDialect = "mysql"
	// SQLDialectPostgres is PostgreSQL.
	SQLDialectPostgres SQLDialect = "postgres"
)

// checkSQLDialect refuses a dialect the SQL functions do not know.
func checkSQLDialect(d SQLDialect) error {
	switch d {
	case SQLDialectSQLite, SQLDialectMySQL, SQLDialectPostgres:
		return nil
	}
	return fmt.Errorf("unknown SQL dialect %q: use SQLDialectSQLite, SQLDialectMySQL or SQLDialectPostgres", string(d))
}

// sqlConn is what both *sql.DB and *sql.Tx offer, so a statement can run
// inside a transaction or on its own.
type sqlConn interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// bindVars writes the ? placeholders of one of insyra's own statements the
// way the dialect expects: $1, $2, … on PostgreSQL, unchanged elsewhere. It
// is for insyra's statements only, which hold no ? inside a literal; a
// caller's ReadSQLOptions.Query goes to the driver as written.
func bindVars(dialect, query string) string {
	if dialect != string(SQLDialectPostgres) || !strings.Contains(query, "?") {
		return query
	}
	var b strings.Builder
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// inSQLTx runs run inside one transaction, committing when it returns nil and
// rolling back otherwise.
func inSQLTx(ctx context.Context, db *sql.DB, run func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := run(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
