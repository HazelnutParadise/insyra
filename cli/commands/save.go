package commands

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/parquet"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "save",
		Usage:       "save <var> <file> | save <var> sql <conn> <table> [if-exists fail|replace|append] [batch N] [schema <s>] [rownames]",
		Description: "Save a DataTable variable to a file or SQL connection",
		Run:         runSaveCommand,
	})
}

func runSaveCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: save <var> <file> | save <var> sql <conn> <table> [...]")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}

	if args[1] == "sql" {
		return runSaveSQL(ctx, args[0], table, args[2:])
	}

	path := args[1]
	switch detectFileKind(path) {
	case "csv":
		err = table.ToCSV(path, false, true, false)
	case "json":
		err = table.ToJSON(path, true)
	case "parquet":
		err = parquet.Write(table, path)
	default:
		return fmt.Errorf("unsupported output file type: %s", path)
	}
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "saved %s to %s\n", args[0], path)
	return nil
}
