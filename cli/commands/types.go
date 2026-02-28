package commands

import (
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "types",
		Usage:       "types <var>",
		Description: "Show value types of DataTable/DataList",
		Run:         runTypesCommand,
	})
}

func runTypesCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: types <var>")
	}
	value, exists := ctx.Vars[args[0]]
	if !exists {
		return fmt.Errorf("variable not found: %s", args[0])
	}
	switch typed := value.(type) {
	case *insyra.DataTable:
		typed.ShowTypes()
	case *insyra.DataList:
		typed.ShowTypes()
	default:
		return fmt.Errorf("types is only supported for DataTable/DataList")
	}
	return nil
}
