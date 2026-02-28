package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "shape",
		Usage:       "shape <var>",
		Description: "Show shape of DataTable/DataList",
		Run:         runShapeCommand,
	})
}

func runShapeCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: shape <var>")
	}
	if table, err := getDataTableVar(ctx, args[0]); err == nil {
		_, _ = fmt.Fprintf(ctx.Output, "(%d, %d)\n", table.NumRows(), table.NumCols())
		return nil
	}
	if list, err := getDataListVar(ctx, args[0]); err == nil {
		_, _ = fmt.Fprintf(ctx.Output, "(%d)\n", list.Len())
		return nil
	}
	return fmt.Errorf("variable %s is not a DataTable/DataList", args[0])
}
