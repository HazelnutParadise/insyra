package commands

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/parquet"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "save",
		Usage:       "save <var> <file>",
		Description: "Save DataTable variable to file",
		Run:         runSaveCommand,
	})
}

func runSaveCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: save <var> <file>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
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
