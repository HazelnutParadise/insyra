package commands

import (
	"fmt"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "parsedates",
		Usage:       "parsedates <var> [cols <c1,c2>] [layout <go-layout>] [as <var>]",
		Description: "Convert date strings to time.Time in a DataList or DataTable columns",
		Forms: []string{
			"parsedates <datalist> [layout <go-layout>] [as <var>]",
			"parsedates <datatable> cols <c1,c2> [layout <go-layout>] [as <var>]",
			"",
			"cols is required for a DataTable; columns may be names or Excel indices (A, B, ...).",
			"layout takes a Go reference layout (2006-01-02) and may be repeated; the first match wins.",
			"Without layout, common ISO layouts are tried. Cells no layout matches become nil.",
		},
		Examples: []string{
			"insyra parsedates bars cols Date as bars",
			"insyra parsedates bars cols Date,Settled layout 02/01/2006 as bars",
			"insyra parsedates dates layout 2006/01/02 as dates",
		},
		Run: runParseDatesCommand,
	})
}

func runParseDatesCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: parsedates <var> [cols <c1,c2>] [layout <go-layout>] [as <var>]")
	}
	cols, layouts, err := parseParseDatesOptions(coreArgs[1:])
	if err != nil {
		return err
	}

	name := coreArgs[0]
	value, exists := ctx.Vars[name]
	if !exists {
		return fmt.Errorf("parsedates: variable not found: %s", name)
	}

	switch source := value.(type) {
	case *insyra.DataList:
		if len(cols) > 0 {
			return fmt.Errorf("parsedates: cols applies to a DataTable; %s is a DataList (the whole list is converted)", name)
		}
		ctx.Vars[alias] = source.Clone().ParseDates(layouts...)
		_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	case *insyra.DataTable:
		if len(cols) == 0 {
			return fmt.Errorf("parsedates: cols is required for a DataTable (parsedates %s cols <c1,c2>)", name)
		}
		ctx.Vars[alias] = source.Clone().ParseDatesCols(cols, layouts...)
		_, _ = fmt.Fprintf(ctx.Output, "saved as %s (%s)\n", alias, strings.Join(cols, ", "))
	default:
		return fmt.Errorf("parsedates: variable %s is not a DataList or DataTable", name)
	}
	return nil
}

// parseParseDatesOptions reads the `cols` / `layout` option pairs. layout may
// repeat, in which case the layouts are tried in the order given.
func parseParseDatesOptions(args []string) (cols []string, layouts []string, err error) {
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("parsedates: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "cols", "col", "columns":
			v, nextErr := next()
			if nextErr != nil {
				return nil, nil, nextErr
			}
			parsed := parseCSVTokens(v)
			if len(parsed) == 0 {
				return nil, nil, fmt.Errorf("parsedates: cols needs at least one column name")
			}
			cols = append(cols, parsed...)
			i += 2
		case "layout", "layouts":
			v, nextErr := next()
			if nextErr != nil {
				return nil, nil, nextErr
			}
			if strings.TrimSpace(v) == "" {
				return nil, nil, fmt.Errorf("parsedates: layout cannot be empty")
			}
			layouts = append(layouts, v)
			i += 2
		default:
			return nil, nil, fmt.Errorf("parsedates: unknown option %q (supported: cols, layout)", args[i])
		}
	}
	return cols, layouts, nil
}
