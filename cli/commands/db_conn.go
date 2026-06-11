package commands

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// DBConn represents a named database connection registered in an ExecContext.
type DBConn struct {
	Name    string
	Dialect string // "sqlite", "mysql", "postgres"
	DSN     string // original DSN string supplied by the user (with password masked when displayed)
	DB      *gorm.DB
}

// openDBConn opens a database connection from a dsn of the form
//
//	sqlite:<path-or-uri>      e.g. sqlite::memory:, sqlite:./foo.db, sqlite:file:./foo.db?mode=ro
//	mysql:<go-sql-driver-dsn> e.g. mysql:user:pass@tcp(host:3306)/db
//	mysql://user:pass@host:port/db?param=value (URL form, auto-converted)
//	postgres://user:pass@host:port/db?sslmode=disable (URL form, native to pgx)
//	postgres:host=... user=... password=... dbname=... (libpq KV form)
//
// All three dialects use pure-Go drivers (modernc.org/sqlite, go-sql-driver/mysql, jackc/pgx).
func openDBConn(name, dsn string) (*DBConn, error) {
	dialect, raw, err := splitDSN(dsn)
	if err != nil {
		return nil, err
	}

	var db *gorm.DB
	switch dialect {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(raw), &gorm.Config{})
	case "mysql":
		db, err = gorm.Open(mysql.Open(toMySQLNativeDSN(raw)), &gorm.Config{})
	case "postgres":
		db, err = gorm.Open(postgres.Open(raw), &gorm.Config{})
	default:
		return nil, fmt.Errorf("unsupported dialect %q (supported: sqlite, mysql, postgres)", dialect)
	}
	if err != nil {
		return nil, fmt.Errorf("opening %s connection: %w", dialect, err)
	}

	// Eagerly verify the connection so the user gets immediate feedback.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("getting underlying *sql.DB: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("pinging %s: %w", dialect, err)
	}

	return &DBConn{
		Name:    name,
		Dialect: dialect,
		DSN:     dsn,
		DB:      db,
	}, nil
}

// splitDSN splits "<dialect>:<rest>" into the dialect prefix and the remainder.
// The dialect comparison is case-insensitive; "postgresql" is normalized to
// "postgres".
func splitDSN(dsn string) (dialect, rest string, err error) {
	idx := strings.Index(dsn, ":")
	if idx <= 0 {
		return "", "", fmt.Errorf("invalid dsn %q: expected <dialect>:<...>", dsn)
	}
	dialect = strings.ToLower(dsn[:idx])
	rest = dsn[idx+1:]
	if dialect == "postgresql" {
		dialect = "postgres"
		// pgx accepts the full URL including the "postgresql://" prefix, so
		// reconstruct it.
		rest = "postgresql:" + rest
	} else if dialect == "postgres" && strings.HasPrefix(rest, "//") {
		// Native postgres URL form needs the prefix back for pgx.
		rest = "postgres:" + rest
	}
	return dialect, rest, nil
}

// toMySQLNativeDSN converts a URL-style MySQL DSN to the native go-sql-driver
// form, leaving native-form DSNs untouched.
//
//	"//user:pass@host:3306/dbname?param=value"
//	    -> "user:pass@tcp(host:3306)/dbname?param=value"
func toMySQLNativeDSN(raw string) string {
	if !strings.HasPrefix(raw, "//") {
		return raw
	}
	u, err := url.Parse("mysql:" + raw)
	if err != nil || u.Host == "" {
		return raw
	}
	user := u.User.Username()
	pass, hasPass := u.User.Password()
	host := u.Host
	dbname := strings.TrimPrefix(u.Path, "/")
	cred := user
	if hasPass {
		cred = user + ":" + pass
	}
	out := fmt.Sprintf("%s@tcp(%s)/%s", cred, host, dbname)
	if u.RawQuery != "" {
		out += "?" + u.RawQuery
	}
	return out
}

// maskedDSN returns a display-safe copy of the DSN with passwords replaced by
// "***".
func (c *DBConn) maskedDSN() string {
	return maskDSNPassword(c.DSN)
}

func maskDSNPassword(dsn string) string {
	// URL-style: <scheme>://user:pass@host/...
	if i := strings.Index(dsn, "://"); i > 0 {
		head := dsn[:i+3]
		tail := dsn[i+3:]
		if at := strings.Index(tail, "@"); at > 0 {
			cred := tail[:at]
			rest := tail[at:]
			if colon := strings.Index(cred, ":"); colon > 0 {
				return head + cred[:colon] + ":***" + rest
			}
		}
		return dsn
	}
	// MySQL native: user:pass@tcp(host:port)/db
	if at := strings.Index(dsn, "@"); at > 0 {
		head := dsn[:at]
		rest := dsn[at:]
		if colon := strings.Index(head, ":"); colon > 0 {
			// head may be "<dialect>:user:pass". Skip the dialect prefix when
			// present so we don't mask the dialect tag.
			parts := strings.SplitN(head, ":", 3)
			if len(parts) == 3 {
				return parts[0] + ":" + parts[1] + ":***" + rest
			}
			return head[:colon] + ":***" + rest
		}
	}
	return dsn
}

// closeDBConn closes the connection's underlying *sql.DB, ignoring any error
// from a closed handle.
func (c *DBConn) closeDBConn() error {
	if c == nil || c.DB == nil {
		return nil
	}
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// getDBConn fetches a registered connection by name.
func getDBConn(ctx *ExecContext, name string) (*DBConn, error) {
	if ctx.DBConns == nil {
		return nil, fmt.Errorf("no database connections; use 'db connect <name> <dsn>' first")
	}
	c, ok := ctx.DBConns[name]
	if !ok {
		return nil, fmt.Errorf("unknown database connection %q", name)
	}
	return c, nil
}

// CloseAllDBConns closes every connection registered on ctx and clears the
// map. Used by the REPL on shutdown.
func CloseAllDBConns(ctx *ExecContext) {
	if ctx == nil || len(ctx.DBConns) == 0 {
		return
	}
	for _, c := range ctx.DBConns {
		_ = c.closeDBConn()
	}
	ctx.DBConns = nil
}
