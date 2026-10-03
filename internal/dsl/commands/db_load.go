package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

// runLoadSQL implements the `load sql ...` subcommand.
//
//	load sql <conn> <table> [where "..."] [order "..."] [limit N] [offset N] [cols "c1,c2"] [as $var]
//	load sql <conn> query "<SQL>" [params <v1> <v2> ...] [as $var]
//
// `as $var` has already been stripped by parseAlias before this is called.
func runLoadSQL(ctx *ExecContext, args []string) (*insyra.DataTable, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: load sql <conn> <table>|query \"<sql>\" [...]")
	}
	conn, err := getDBConn(ctx, args[0])
	if err != nil {
		return nil, err
	}

	if strings.EqualFold(args[1], "query") {
		if len(args) < 3 {
			return nil, fmt.Errorf("usage: load sql <conn> query \"<sql>\" [params <v1> <v2> ...]")
		}
		opts, err := parseLoadSQLQueryOptions(args[3:])
		if err != nil {
			return nil, err
		}
		opts.Query = args[2]
		return insyra.ReadSQLContext(context.Background(), conn.DB, "", opts)
	}

	table := args[1]
	opts, err := parseLoadSQLTableOptions(args[2:])
	if err != nil {
		return nil, err
	}
	return insyra.ReadSQLContext(context.Background(), conn.DB, table, opts)
}

func parseLoadSQLTableOptions(args []string) (insyra.ReadSQLOptions, error) {
	var opts insyra.ReadSQLOptions
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("load sql: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "where":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.WhereClause = v
			i += 2
		case "order", "orderby":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.OrderBy = v
			i += 2
		case "limit":
			v, err := next()
			if err != nil {
				return opts, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return opts, fmt.Errorf("load sql: invalid limit %q", v)
			}
			opts.Limit = n
			i += 2
		case "offset":
			v, err := next()
			if err != nil {
				return opts, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return opts, fmt.Errorf("load sql: invalid offset %q", v)
			}
			opts.Offset = n
			i += 2
		case "cols", "columns":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Columns = parseCSVTokens(v)
			i += 2
		case "schema":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Schema = v
			i += 2
		case "indexcol", "index":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.IndexCol = v
			i += 2
		case "parsedates":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.ParseDates = parseCSVTokens(v)
			i += 2
		default:
			return opts, fmt.Errorf("load sql: unknown option %q", args[i])
		}
	}
	return opts, nil
}

// parseLoadSQLQueryOptions accepts only `params <v1> <v2> ...` after
// `load sql <conn> query "<SQL>"`. Everything from `params` onwards is consumed
// as positional bind values, parsed with parseLiteral.
func parseLoadSQLQueryOptions(args []string) (insyra.ReadSQLOptions, error) {
	var opts insyra.ReadSQLOptions
	if len(args) == 0 {
		return opts, nil
	}
	if !strings.EqualFold(args[0], "params") {
		return opts, fmt.Errorf("load sql query: unknown option %q (only 'params <v1> <v2> ...' is supported)", args[0])
	}
	rest := args[1:]
	if len(rest) == 0 {
		return opts, fmt.Errorf("load sql query: 'params' requires at least one value")
	}
	opts.Params = make([]any, len(rest))
	for i, raw := range rest {
		opts.Params[i] = parseLiteral(raw)
	}
	return opts, nil
}
