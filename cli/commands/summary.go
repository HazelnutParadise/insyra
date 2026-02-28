package commands

import (
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "summary",
		Usage:       "summary <var>",
		Description: "Show summary statistics",
		Run:         runSummaryCommand,
	})
}

func runSummaryCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: summary <var>")
	}
	value, exists := ctx.Vars[args[0]]
	if !exists {
		return fmt.Errorf("variable not found: %s", args[0])
	}
	switch typed := value.(type) {
	case *insyra.DataTable:
		typed.Summary()
	case *insyra.DataList:
		typed.Summary()
	default:
		return fmt.Errorf("summary is only supported for DataTable/DataList")
	}
	return nil
}
