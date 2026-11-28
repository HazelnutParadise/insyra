package sqlio

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/prep"
)

// Dialect names the database behind the *sql.DB passed to Read, Query,
// Stream, StreamQuery and Write. It decides how identifiers are quoted, how column types are
// inferred, how insyra's own statements bind their values, and how Replace
// swaps a table. A *sql.DB does not say which database it talks to, so the
// caller names it; there is no default.
type Dialect string

const (
	// SQLite is SQLite, through any database/sql driver for it.
	SQLite Dialect = "sqlite"
	// MySQL is MySQL, and servers that speak its dialect.
	MySQL Dialect = "mysql"
	// Postgres is PostgreSQL.
	Postgres Dialect = "postgres"
)

// checkDialect refuses a dialect this package does not know.
func checkDialect(d Dialect) error {
	switch d {
	case SQLite, MySQL, Postgres:
		return nil
	}
	return fmt.Errorf("unknown SQL dialect %q: use sqlio.SQLite, sqlio.MySQL or sqlio.Postgres", string(d))
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
// caller's query goes to the driver as written.
func bindVars(dialect, query string) string {
	if dialect != string(Postgres) || !strings.Contains(query, "?") {
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

// oneOption returns the single options value given, the zero value when none
// is, and an error naming the type when there are more.
func oneOption[T any](typeName string, opts []T) (T, error) {
	var zero T
	if msg := prep.ExtraOptional(typeName, len(opts)); msg != "" {
		return zero, errors.New(msg)
	}
	if len(opts) == 1 {
		return opts[0], nil
	}
	return zero, nil
}
