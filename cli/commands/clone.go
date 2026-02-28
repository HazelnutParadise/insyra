package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "clone",
		Usage:       "clone <var> [as <var>]",
		Description: "Deep clone DataTable/DataList variable",
		Run:         runCloneCommand,
	})
}

func runCloneCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: clone <var> [as <var>]")
	}
	name := coreArgs[0]
	if table, err := getDataTableVar(ctx, name); err == nil {
		ctx.Vars[alias] = table.Clone()
		_, _ = fmt.Fprintf(ctx.Output, "saved clone as %s\n", alias)
		return nil
	}
	if list, err := getDataListVar(ctx, name); err == nil {
		ctx.Vars[alias] = list.Clone()
		_, _ = fmt.Fprintf(ctx.Output, "saved clone as %s\n", alias)
		return nil
	}
	return fmt.Errorf("variable not found: %s", name)
}
