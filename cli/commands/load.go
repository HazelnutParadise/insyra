package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parquet"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "load",
		Usage:       "load <file> [headers true|false] [rownames true|false] [encoding <enc>] [sheet <name>] | load parquet <file> [...] | load sql <conn> <table>|query \"<sql>\" [...] [as <var>]",
		Description: "Load data into a DataTable variable from a file, parquet, or SQL connection",
		Forms: []string{
			"load <file.csv> [headers true|false] [rownames true|false] [encoding <enc>] [as <var>]",
			"load <file.json> [as <var>]",
			"load <file.xlsx> sheet <name> [headers true|false] [rownames true|false] [as <var>]",
			"load parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] [as <var>]",
			"load sql <conn> <table> [where \"...\"] [order \"...\"] [limit N] [offset N] [cols \"c1,c2\"] [schema <s>] [indexcol <c>] [parsedates \"c1,c2\"] [as <var>]",
			"load sql <conn> query \"<SQL>\" [params <v1> <v2> ...] [as <var>]",
			"",
			"File option defaults: headers=true, rownames=false.",
			"Booleans accept true|false|yes|no|on|off|1|0.",
		},
		Examples: []string{
			"insyra load sales.csv as t",
			"insyra load matrix.csv headers false as raw",
			"insyra load gdp.csv rownames true as gdp",
			"insyra load legacy.csv encoding big5 as legacy",
			"insyra load report.xlsx sheet 2025 rownames true as r",
			"insyra load parquet data.parquet cols id,amount rowgroups 0,1 as p",
			"insyra load sql main customers as customers",
			"insyra load sql main query \"SELECT * FROM orders WHERE year = ?\" params 2025 as orders",
		},
		Run: runLoadCommand,
	})
}

func runLoadCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) == 0 {
		return fmt.Errorf("usage: load <file> [headers true|false] [rownames true|false] [encoding <enc>] [sheet <name>] | load parquet <file> [...] | load sql <conn> <table>|query \"<sql>\" [...] [as <var>]")
	}

	var table *insyra.DataTable
	var err error

	switch coreArgs[0] {
	case "parquet":
		if len(coreArgs) < 2 {
			return fmt.Errorf("usage: load parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] [as <var>]")
		}
		opt, parseErr := parseParquetLoadOptions(coreArgs[2:])
		if parseErr != nil {
			return parseErr
		}
		table, err = parquet.Read(context.Background(), coreArgs[1], opt)
		if err != nil {
			return err
		}
	case "sql":
		table, err = runLoadSQL(ctx, coreArgs[1:])
		if err != nil {
			return err
		}
	default:
		path := coreArgs[0]
		opts, parseErr := parseFileLoadOptions(coreArgs[1:])
		if parseErr != nil {
			return parseErr
		}
		switch detectFileKind(path) {
		case "csv":
			if opts.SheetSet {
				return fmt.Errorf("load csv: 'sheet' is not valid for CSV files")
			}
			if opts.Encoding != "" {
				table, err = insyra.ReadCSV_File(path, opts.RowNames, opts.Headers, opts.Encoding)
			} else {
				table, err = insyra.ReadCSV_File(path, opts.RowNames, opts.Headers)
			}
		case "json":
			if opts.HeadersSet || opts.RowNamesSet || opts.SheetSet || opts.Encoding != "" {
				return fmt.Errorf("load json: headers/rownames/sheet/encoding options are not supported for JSON")
			}
			table, err = insyra.ReadJSON_File(path)
		case "excel":
			if opts.Encoding != "" {
				return fmt.Errorf("load excel: 'encoding' is not valid for Excel files")
			}
			if !opts.SheetSet || opts.Sheet == "" {
				return fmt.Errorf("usage for excel: load <file.xlsx> sheet <sheet-name> [headers true|false] [rownames true|false] [as <var>]")
			}
			table, err = insyra.ReadExcelSheet(path, opts.Sheet, opts.RowNames, opts.Headers)
		default:
			return fmt.Errorf("unsupported file type: %s", path)
		}
		if err != nil {
			return err
		}
	}

	ctx.Vars[alias] = table
	_, _ = fmt.Fprintf(ctx.Output, "loaded %s (rows=%d, cols=%d)\n", alias, table.NumRows(), table.NumCols())
	return nil
}

// fileLoadOptions captures the shared CSV/Excel/JSON load options. The *Set
// flags let format-specific code reject options that don't apply.
type fileLoadOptions struct {
	Headers     bool
	HeadersSet  bool
	RowNames    bool
	RowNamesSet bool
	Encoding    string
	Sheet       string
	SheetSet    bool
}

func parseFileLoadOptions(args []string) (fileLoadOptions, error) {
	opts := fileLoadOptions{Headers: true, RowNames: false}
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("load: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "headers", "header":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("load: invalid value for headers: %w", err)
			}
			opts.Headers = b
			opts.HeadersSet = true
			i += 2
		case "rownames", "rowname":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("load: invalid value for rownames: %w", err)
			}
			opts.RowNames = b
			opts.RowNamesSet = true
			i += 2
		case "encoding":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Encoding = v
			i += 2
		case "sheet":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Sheet = v
			opts.SheetSet = true
			i += 2
		default:
			return opts, fmt.Errorf("load: unknown option %q (supported: headers, rownames, encoding, sheet)", args[i])
		}
	}
	return opts, nil
}

func parseParquetLoadOptions(args []string) (parquet.ReadOptions, error) {
	opt := parquet.ReadOptions{}
	seenCols := false
	seenRowGroups := false

	for index := 0; index < len(args); {
		switch strings.ToLower(args[index]) {
		case "cols":
			if seenCols {
				return parquet.ReadOptions{}, fmt.Errorf("load parquet: cols option provided more than once")
			}
			if index+1 >= len(args) {
				return parquet.ReadOptions{}, fmt.Errorf("usage: load parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] [as <var>]")
			}
			cols := parseCSVTokens(args[index+1])
			if len(cols) == 0 {
				return parquet.ReadOptions{}, fmt.Errorf("load parquet: cols requires at least one column name")
			}
			opt.Columns = cols
			seenCols = true
			index += 2
		case "rowgroups":
			if seenRowGroups {
				return parquet.ReadOptions{}, fmt.Errorf("load parquet: rowgroups option provided more than once")
			}
			if index+1 >= len(args) {
				return parquet.ReadOptions{}, fmt.Errorf("usage: load parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] [as <var>]")
			}
			groups, err := parseCSVInts(args[index+1])
			if err != nil {
				return parquet.ReadOptions{}, err
			}
			if len(groups) == 0 {
				return parquet.ReadOptions{}, fmt.Errorf("load parquet: rowgroups requires at least one index")
			}
			opt.RowGroups = groups
			seenRowGroups = true
			index += 2
		default:
			return parquet.ReadOptions{}, fmt.Errorf("load parquet: unknown option %q (supported: cols, rowgroups)", args[index])
		}
	}

	return opt, nil
}

func parseCSVTokens(raw string) []string {
	tokens := strings.Split(raw, ",")
	cleaned := make([]string, 0, len(tokens))
	for _, token := range tokens {
		value := strings.TrimSpace(token)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	return cleaned
}

func parseCSVInts(raw string) ([]int, error) {
	tokens := parseCSVTokens(raw)
	values := make([]int, 0, len(tokens))
	for _, token := range tokens {
		value, err := strconv.Atoi(token)
		if err != nil {
			return nil, fmt.Errorf("load parquet: invalid rowgroup index %q", token)
		}
		if value < 0 {
			return nil, fmt.Errorf("load parquet: rowgroup index must be >= 0, got %d", value)
		}
		values = append(values, value)
	}
	return values, nil
}
