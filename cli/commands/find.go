package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "find",
		Usage:       "find <var> <value>",
		Description: "Find rows containing value",
		Run:         runFindCommand,
	})
}

func runFindCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: find <var> <value>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	rows := table.FindRowsIfContains(parseLiteral(args[1]))
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", rows)
	return nil
}
