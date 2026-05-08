package commands

import (
	"fmt"
	"sort"
	"strings"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "db",
		Usage:       "db connect <name> <dsn> | db list | db tables <name> [schema <s>] | db disconnect <name>",
		Description: "Manage named database connections (sqlite, mysql, postgres; pure-Go drivers)",
		Forms: []string{
			"db connect <name> <dsn>                     open and register a named connection",
			"db list                                     list active connections (passwords masked)",
			"db tables <name> [schema <s>]               list tables on a connection",
			"db disconnect <name>                        close and unregister a connection",
			"",
			"DSN forms:",
			"  sqlite:<path-or-uri>                      e.g. sqlite::memory:, sqlite:./foo.db",
			"  mysql:<go-sql-driver-dsn>                 e.g. mysql:user:pass@tcp(host:3306)/db",
			"  mysql://user:pass@host:port/db?...        URL form, auto-converted",
			"  postgres://user:pass@host:port/db?...     pgx URL form",
			"  postgres:host=... user=... dbname=...     libpq KV form",
		},
		Examples: []string{
			"insyra db connect main sqlite:./demo.db",
			"insyra db tables main schema public",
			"insyra db list",
			"insyra db disconnect main",
		},
		Run: runDBCommand,
	})
}

func runDBCommand(ctx *ExecContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: db connect <name> <dsn> | db list | db tables <name> [schema <s>] | db disconnect <name>")
	}
	switch strings.ToLower(args[0]) {
	case "connect":
		return runDBConnect(ctx, args[1:])
	case "list", "ls":
		return runDBList(ctx, args[1:])
	case "tables":
		return runDBTables(ctx, args[1:])
	case "disconnect", "close":
		return runDBDisconnect(ctx, args[1:])
	default:
		return fmt.Errorf("unknown db subcommand %q (expected: connect, list, tables, disconnect)", args[0])
	}
}

func runDBConnect(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: db connect <name> <dsn>")
	}
	name := args[0]
	dsn := strings.Join(args[1:], " ") // tolerate spaces inside DSN (e.g. libpq KV form)
	if name == "" {
		return fmt.Errorf("connection name cannot be empty")
	}
	if ctx.DBConns == nil {
		ctx.DBConns = make(map[string]*DBConn)
	}
	if _, exists := ctx.DBConns[name]; exists {
		return fmt.Errorf("connection %q already exists; disconnect it first", name)
	}
	conn, err := openDBConn(name, dsn)
	if err != nil {
		return err
	}
	ctx.DBConns[name] = conn
	_, _ = fmt.Fprintf(ctx.Output, "connected %s (%s)\n", name, conn.Dialect)
	return nil
}

func runDBList(ctx *ExecContext, _ []string) error {
	if len(ctx.DBConns) == 0 {
		_, _ = fmt.Fprintln(ctx.Output, "(no database connections)")
		return nil
	}
	names := make([]string, 0, len(ctx.DBConns))
	for n := range ctx.DBConns {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		c := ctx.DBConns[n]
		_, _ = fmt.Fprintf(ctx.Output, "%s\t%s\t%s\n", c.Name, c.Dialect, c.maskedDSN())
	}
	return nil
}

func runDBDisconnect(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: db disconnect <name>")
	}
	name := args[0]
	conn, err := getDBConn(ctx, name)
	if err != nil {
		return err
	}
	if cerr := conn.closeDBConn(); cerr != nil {
		return fmt.Errorf("closing %s: %w", name, cerr)
	}
	delete(ctx.DBConns, name)
	_, _ = fmt.Fprintf(ctx.Output, "disconnected %s\n", name)
	return nil
}

func runDBTables(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: db tables <name> [schema <s>]")
	}
	conn, err := getDBConn(ctx, args[0])
	if err != nil {
		return err
	}
	schema := ""
	for i := 1; i < len(args); {
		if strings.EqualFold(args[i], "schema") {
			if i+1 >= len(args) {
				return fmt.Errorf("db tables: schema requires a value")
			}
			schema = args[i+1]
			i += 2
			continue
		}
		return fmt.Errorf("db tables: unknown option %q", args[i])
	}

	tables, err := listTables(conn, schema)
	if err != nil {
		return err
	}
	if len(tables) == 0 {
		_, _ = fmt.Fprintln(ctx.Output, "(no tables)")
		return nil
	}
	for _, name := range tables {
		_, _ = fmt.Fprintln(ctx.Output, name)
	}
	return nil
}

// listTables returns the table names visible on the connection. For Postgres
// and MySQL, an explicit schema argument overrides the default (current schema
// / current database).
func listTables(conn *DBConn, schema string) ([]string, error) {
	var (
		query string
		args  []any
	)
	switch conn.Dialect {
	case "sqlite":
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
	case "mysql":
		if schema != "" {
			query = "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_SCHEMA = ? ORDER BY TABLE_NAME"
			args = []any{schema}
		} else {
			query = "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' AND TABLE_SCHEMA = (SELECT DATABASE()) ORDER BY TABLE_NAME"
		}
	case "postgres":
		if schema != "" {
			query = "SELECT table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema = ? ORDER BY table_name"
			args = []any{schema}
		} else {
			query = "SELECT table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema = current_schema() ORDER BY table_name"
		}
	default:
		return nil, fmt.Errorf("db tables: unsupported dialect %q", conn.Dialect)
	}

	rows, err := conn.DB.Raw(query, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("listing tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
