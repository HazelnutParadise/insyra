package commands

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "describe",
		Usage:       "describe <var> [by <col1>[,<col2>...]] [all true|false] [percentiles <p1,p2,...>] [as <var>]",
		Description: "Create a programmatic summary table",
		Forms: []string{
			"describe <var> [as <var>]",
			"describe <var> all true|false [as <var>]",
			"describe <var> percentiles <p1,p2,...> [as <var>]",
			"describe <var> by <col1>[,<col2>...] [all true|false] [percentiles <p1,p2,...>] [as <var>]",
		},
		Examples: []string{
			"insyra describe sales as summary",
			"insyra describe sales all true as summary",
			"insyra describe sales percentiles 0.1,0.5,0.9 as summary",
			"insyra describe sales by region all true as by_region",
		},
		Run: runDescribeCommand,
	})
}

type describeCommandOptions struct {
	insyra.DescribeOptions
	GroupBy []string
}

func runDescribeCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: describe <var> [by <cols>] [all true|false] [percentiles <p1,p2,...>] [as <var>]")
	}
	varName := coreArgs[0]
	opts, err := parseDescribeCommandOptions(coreArgs[1:])
	if err != nil {
		return err
	}
	value, exists := ctx.Vars[varName]
	if !exists {
		return fmt.Errorf("variable not found: %s", varName)
	}

	var result *insyra.DataTable
	switch typed := value.(type) {
	case *insyra.DataTable:
		if len(opts.GroupBy) > 0 {
			result = typed.GroupBy(opts.GroupBy...).Describe(opts.DescribeOptions)
			if errInfo := typed.Err(); errInfo != nil {
				_, _ = fmt.Fprintf(ctx.Output, "warning: %s\n", errInfo.Error())
			}
		} else {
			result = typed.Describe(opts.DescribeOptions)
		}
	case *insyra.DataList:
		if len(opts.GroupBy) > 0 {
			return fmt.Errorf("describe: by is only supported for DataTable variables")
		}
		result = typed.Describe(opts.DescribeOptions)
	default:
		return fmt.Errorf("describe is only supported for DataTable/DataList")
	}
	ctx.Vars[alias] = result
	result.Show()
	_, _ = fmt.Fprintf(ctx.Output, "saved description as %s\n", alias)
	return nil
}

func parseDescribeCommandOptions(args []string) (describeCommandOptions, error) {
	var opts describeCommandOptions
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		if i+1 >= len(args) {
			return opts, fmt.Errorf("describe: option %q requires a value", args[i])
		}
		value := args[i+1]
		switch key {
		case "by":
			cols := parseDescribeColumns(value)
			if len(cols) == 0 {
				return opts, fmt.Errorf("describe: by requires at least one column")
			}
			opts.GroupBy = cols
		case "all":
			parsed, err := parseFlexBool(value)
			if err != nil {
				return opts, fmt.Errorf("describe: invalid value for all: %w", err)
			}
			opts.IncludeAll = parsed
		case "percentiles":
			ps, err := parseDescribePercentiles(value)
			if err != nil {
				return opts, err
			}
			opts.Percentiles = ps
		default:
			return opts, fmt.Errorf("describe: unknown option %q (supported: by, all, percentiles)", args[i])
		}
		i += 2
	}
	return opts, nil
}

func parseDescribeColumns(raw string) []string {
	tokens := strings.Split(raw, ",")
	cols := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token != "" {
			cols = append(cols, token)
		}
	}
	return cols
}

func parseDescribePercentiles(raw string) ([]float64, error) {
	tokens := strings.Split(raw, ",")
	ps := make([]float64, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		p, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return nil, fmt.Errorf("describe: invalid percentile %q", token)
		}
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 1 {
			return nil, fmt.Errorf("describe: percentile %v out of range [0, 1]", p)
		}
		ps = append(ps, p)
	}
	if len(ps) == 0 {
		return nil, fmt.Errorf("describe: percentiles requires at least one value")
	}
	return ps, nil
}
