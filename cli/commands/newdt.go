package commands

import (
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "newdt",
		Usage:       "newdt <dl_vars...> [as <var>]",
		Description: "Create DataTable from DataList variables",
		Run:         runNewDTCommand,
	})
}

func runNewDTCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) == 0 {
		dt := insyra.NewDataTable()
		ctx.Vars[alias] = dt
		_, _ = fmt.Fprintf(ctx.Output, "created empty datatable %s\n", alias)
		return nil
	}

	columns := make([]*insyra.DataList, 0, len(coreArgs))
	for _, name := range coreArgs {
		dl, err := getDataListVar(ctx, name)
		if err != nil {
			return err
		}
		columns = append(columns, dl)
	}
	dt := insyra.NewDataTable(columns...)
	ctx.Vars[alias] = dt
	_, _ = fmt.Fprintf(ctx.Output, "created datatable %s (rows=%d, cols=%d)\n", alias, dt.NumRows(), dt.NumCols())
	return nil
}
