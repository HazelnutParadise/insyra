package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "cols",
		Usage:       "cols <var>",
		Description: "List DataTable column names",
		Run:         runColsCommand,
	})
}

func runColsCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cols <var>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	for _, colName := range table.ColNames() {
		_, _ = fmt.Fprintln(ctx.Output, colName)
	}
	return nil
}
