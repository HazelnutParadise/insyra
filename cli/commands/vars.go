package commands

import (
	"fmt"
	"reflect"
	"sort"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "vars",
		Usage:       "vars",
		Description: "List variables in current environment",
		Run:         runVarsCommand,
	})
}

func runVarsCommand(ctx *ExecContext, args []string) error {
	if len(ctx.Vars) == 0 {
		_, _ = fmt.Fprintln(ctx.Output, "no variables")
		return nil
	}
	keys := make([]string, 0, len(ctx.Vars))
	for key := range ctx.Vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := ctx.Vars[key]
		typeName := "unknown"
		if value != nil {
			typeName = reflect.TypeOf(value).String()
		}
		_, _ = fmt.Fprintf(ctx.Output, "%s\t%s\n", key, typeName)
	}
	return nil
}
