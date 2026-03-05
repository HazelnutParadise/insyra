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
		Usage:       "load <file>|parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] [sheet <name>] [as <var>]",
		Description: "Load data file into DataTable variable",
		Run:         runLoadCommand,
	})
}

func runLoadCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) == 0 {
		return fmt.Errorf("usage: load <file>|parquet <file> [cols <c1,c2,...>] [rowgroups <i1,i2,...>] [sheet <name>] [as <var>]")
	}

	var table *insyra.DataTable
	var err error

	if coreArgs[0] == "parquet" {
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
	} else {
		path := coreArgs[0]
		switch detectFileKind(path) {
		case "csv":
			table, err = insyra.ReadCSV_File(path, false, true)
		case "json":
			table, err = insyra.ReadJSON_File(path)
		case "excel":
			sheetName := ""
			if len(coreArgs) >= 3 && coreArgs[1] == "sheet" {
				sheetName = coreArgs[2]
			}
			if sheetName == "" {
				return fmt.Errorf("usage for excel: load <file.xlsx> sheet <sheet-name> [as <var>]")
			}
			table, err = insyra.ReadExcelSheet(path, sheetName, false, true)
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
