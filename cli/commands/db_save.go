package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

// runSaveSQL implements the `save <var> sql ...` subcommand.
//
//	save <var> sql <conn> <table> [if-exists fail|replace|append] [batch N] [schema <s>] [rownames]
//
// `var` is the variable name (passed in as varName for log messages); table is
// the already-resolved DataTable. args holds the remainder after `sql`.
func runSaveSQL(ctx *ExecContext, varName string, table *insyra.DataTable, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: save <var> sql <conn> <table> [if-exists fail|replace|append] [batch N] [schema <s>] [rownames]")
	}
	conn, err := getDBConn(ctx, args[0])
	if err != nil {
		return err
	}
	tableName := args[1]
	opts, err := parseSaveSQLOptions(args[2:])
	if err != nil {
		return err
	}
	if err := table.ToSQLContext(context.Background(), conn.DB, tableName, opts); err != nil {
		return err
	}
	rows, _ := table.Size()
	_, _ = fmt.Fprintf(ctx.Output, "saved %s to %s.%s (%d rows)\n", varName, args[0], tableName, rows)
	return nil
}

func parseSaveSQLOptions(args []string) (insyra.ToSQLOptions, error) {
	var opts insyra.ToSQLOptions
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("save sql: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "if-exists", "ifexists":
			v, err := next()
			if err != nil {
				return opts, err
			}
			switch strings.ToLower(v) {
			case "fail":
				opts.IfExists = insyra.SQLActionIfTableExistsFail
			case "replace":
				opts.IfExists = insyra.SQLActionIfTableExistsReplace
			case "append":
				opts.IfExists = insyra.SQLActionIfTableExistsAppend
			default:
				return opts, fmt.Errorf("save sql: invalid if-exists %q (expected fail|replace|append)", v)
			}
			i += 2
		case "batch", "batchsize":
			v, err := next()
			if err != nil {
				return opts, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return opts, fmt.Errorf("save sql: invalid batch size %q", v)
			}
			opts.BatchSize = n
			i += 2
		case "schema":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Schema = v
			i += 2
		case "rownames":
			// flag, no value
			opts.RowNames = true
			i++
		default:
			return opts, fmt.Errorf("save sql: unknown option %q", args[i])
		}
	}
	return opts, nil
}
