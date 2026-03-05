package commands

import (
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "newdl",
		Usage:       "newdl <values...> [as <var>]",
		Description: "Create DataList manually",
		Run:         runNewDLCommand,
	})
}

func runNewDLCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	values := make([]any, 0, len(coreArgs))
	for _, raw := range coreArgs {
		values = append(values, parseLiteral(raw))
	}
	dl := insyra.NewDataList(values...)
	ctx.Vars[alias] = dl
	_, _ = fmt.Fprintf(ctx.Output, "created datalist %s (len=%d)\n", alias, dl.Len())
	return nil
}
