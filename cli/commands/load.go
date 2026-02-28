package commands

import (
	"context"
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/parquet"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "load",
		Usage:       "load <file>|parquet <file> [sheet <name>] [as <var>]",
		Description: "Load data file into DataTable variable",
		Run:         runLoadCommand,
	})
}

func runLoadCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) == 0 {
		return fmt.Errorf("usage: load <file>|parquet <file> [sheet <name>] [as <var>]")
	}

	var table *insyra.DataTable
	var err error

	if coreArgs[0] == "parquet" {
		if len(coreArgs) < 2 {
			return fmt.Errorf("usage: load parquet <file> [as <var>]")
		}
		table, err = parquet.Read(context.Background(), coreArgs[1], parquet.ReadOptions{})
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
