package commands

import (
	"fmt"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "pivot",
		Usage:       "pivot <var> index <col1[,col2,...]> columns <col> values <col> [agg <op>] [fillna <literal>] [sortcols true|false] [as <var>]",
		Description: "Reshape long-form DataTable to wide form (long -> wide)",
		Forms: []string{
			"pivot <var> index <cols> columns <col> values <col>          unique (Index, Columns) required",
			"pivot <var> ... agg <op>                                     aggregate duplicate (Index, Columns)",
			"",
			"Ops: sum, mean (alias avg), median, min, max,",
			"     count (non-nil), countall (group size),",
			"     stdev (alias std), stdevp (alias stdp), var, varp,",
			"     first, last, nunique",
		},
		Examples: []string{
			"insyra pivot sales index region columns product values amount as wide",
			"insyra pivot sales index region columns product values amount agg sum fillna 0 sortcols true as wide",
			"insyra pivot orders index region,year columns product values revenue agg mean",
		},
		Run: runPivotCommand,
	})

	_ = Register(&CommandHandler{
		Name:        "unpivot",
		Usage:       "unpivot <var> idvars <col1[,col2,...]> [valuevars <col1[,col2,...]>] [varname <name>] [valuename <name>] [dropna true|false] [as <var>]",
		Description: "Reshape wide-form DataTable to long form (wide -> long)",
		Forms: []string{
			"unpivot <var> idvars <cols>                                  defaults: all non-id cols, varname=variable, valuename=value",
			"unpivot <var> idvars <cols> valuevars <cols>                 explicit value columns",
			"unpivot <var> ... dropna true                                skip nil/NaN values",
		},
		Examples: []string{
			"insyra unpivot survey idvars id valuevars Q1,Q2,Q3 varname question valuename score as long",
			"insyra unpivot wide idvars id as long",
			"insyra unpivot wide idvars id dropna true as long",
		},
		Run: runUnpivotCommand,
	})
}

type pivotOptions struct {
	Index    []string
	Columns  string
	Values   string
	Agg      string
	FillNA   any
	SortCols bool
}

func runPivotCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: pivot <var> index <cols> columns <col> values <col> [agg <op>] [fillna <literal>] [sortcols true|false] [as <var>]")
	}
	varName := coreArgs[0]
	opts, err := parsePivotOptions(coreArgs[1:])
	if err != nil {
		return err
	}
	if len(opts.Index) == 0 {
		return fmt.Errorf("pivot: index is required (use 'index <col1[,col2,...]>')")
	}
	if opts.Columns == "" {
		return fmt.Errorf("pivot: columns is required (use 'columns <col>')")
	}
	if opts.Values == "" {
		return fmt.Errorf("pivot: values is required (use 'values <col>')")
	}

	table, err := getDataTableVar(ctx, varName)
	if err != nil {
		return err
	}
	result, perr := table.Pivot(insyra.PivotConfig{
		Index:    opts.Index,
		Columns:  opts.Columns,
		Values:   opts.Values,
		AggFunc:  opts.Agg,
		FillNA:   opts.FillNA,
		SortCols: opts.SortCols,
	})
	if perr != nil {
		return perr
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "pivoted into %s (%d rows, %d cols)\n", alias, result.NumRows(), result.NumCols())
	return nil
}

func parsePivotOptions(args []string) (pivotOptions, error) {
	var opts pivotOptions
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("pivot: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "index":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Index = parseCSVTokens(v)
			i += 2
		case "columns":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Columns = v
			i += 2
		case "values":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Values = v
			i += 2
		case "agg":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Agg = v
			i += 2
		case "fillna":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.FillNA = parseLiteral(v)
			i += 2
		case "sortcols":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("pivot: invalid value for sortcols: %w", err)
			}
			opts.SortCols = b
			i += 2
		default:
			return opts, fmt.Errorf("pivot: unknown option %q (supported: index, columns, values, agg, fillna, sortcols)", args[i])
		}
	}
	return opts, nil
}

type unpivotOptions struct {
	IDVars    []string
	ValueVars []string
	VarName   string
	ValueName string
	DropNA    bool
}

func runUnpivotCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: unpivot <var> idvars <cols> [valuevars <cols>] [varname <name>] [valuename <name>] [dropna true|false] [as <var>]")
	}
	varName := coreArgs[0]
	opts, err := parseUnpivotOptions(coreArgs[1:])
	if err != nil {
		return err
	}
	if len(opts.IDVars) == 0 {
		return fmt.Errorf("unpivot: idvars is required (use 'idvars <col1[,col2,...]>')")
	}

	table, err := getDataTableVar(ctx, varName)
	if err != nil {
		return err
	}
	result, perr := table.Unpivot(insyra.UnpivotConfig{
		IDVars:    opts.IDVars,
		ValueVars: opts.ValueVars,
		VarName:   opts.VarName,
		ValueName: opts.ValueName,
		DropNA:    opts.DropNA,
	})
	if perr != nil {
		return perr
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "unpivoted into %s (%d rows)\n", alias, result.NumRows())
	return nil
}

func parseUnpivotOptions(args []string) (unpivotOptions, error) {
	var opts unpivotOptions
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("unpivot: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "idvars":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.IDVars = parseCSVTokens(v)
			i += 2
		case "valuevars":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.ValueVars = parseCSVTokens(v)
			i += 2
		case "varname":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.VarName = v
			i += 2
		case "valuename":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.ValueName = v
			i += 2
		case "dropna":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("unpivot: invalid value for dropna: %w", err)
			}
			opts.DropNA = b
			i += 2
		default:
			return opts, fmt.Errorf("unpivot: unknown option %q (supported: idvars, valuevars, varname, valuename, dropna)", args[i])
		}
	}
	return opts, nil
}
