package commands

import (
	"fmt"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "groupby",
		Usage:       "groupby <var> by <col1>[,<col2>...] agg <col>:<op>[:<alias>] [<col>:<op>[:<alias>] ...] [as <var>]",
		Description: "Group a DataTable and aggregate columns",
		Run:         runGroupByCommand,
	})
}

func runGroupByCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 5 {
		return fmt.Errorf("usage: groupby <var> by <cols> agg <col>:<op>[:<alias>] [...] [as <var>]")
	}
	if !strings.EqualFold(coreArgs[1], "by") {
		return fmt.Errorf("expected 'by' after variable name, got %q", coreArgs[1])
	}
	keyTokens := strings.Split(coreArgs[2], ",")
	keys := make([]string, 0, len(keyTokens))
	for _, k := range keyTokens {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return fmt.Errorf("groupby requires at least one key column after 'by'")
	}
	if !strings.EqualFold(coreArgs[3], "agg") {
		return fmt.Errorf("expected 'agg' after key columns, got %q", coreArgs[3])
	}
	specs := coreArgs[4:]
	if len(specs) == 0 {
		return fmt.Errorf("groupby requires at least one aggregate spec after 'agg'")
	}
	configs := make([]insyra.AggregateConfig, 0, len(specs))
	for _, spec := range specs {
		cfg, err := parseAggregateSpec(spec)
		if err != nil {
			return err
		}
		configs = append(configs, cfg)
	}

	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	result := table.GroupBy(keys...).Aggregate(configs...)
	if errInfo := table.Err(); errInfo != nil {
		// Surface the parent-level error to the user. The result is still
		// stored so they can inspect partial output.
		_, _ = fmt.Fprintf(ctx.Output, "warning: %s\n", errInfo.Error())
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "grouped into %s (%d rows)\n", alias, result.NumRows())
	return nil
}

// parseAggregateSpec parses a single CLI aggregate descriptor of the form
//
//	<col>:<op>[:<alias>]
//
// or, for the row-count operator that needs no source column,
//
//	:countall[:<alias>]
//	count          (special-case shorthand for :countall:count)
//
// Aliases are auto-derived as "<col>_<op>" when omitted.
func parseAggregateSpec(spec string) (insyra.AggregateConfig, error) {
	var cfg insyra.AggregateConfig
	if strings.EqualFold(spec, "count") {
		// shorthand: total row count, no source column
		cfg.Op = insyra.OpCountAll
		cfg.As = "count"
		return cfg, nil
	}
	parts := strings.SplitN(spec, ":", 3)
	if len(parts) < 2 {
		return cfg, fmt.Errorf("invalid aggregate spec %q (expected <col>:<op>[:<alias>])", spec)
	}
	cfg.SourceCol = strings.TrimSpace(parts[0])
	op, err := parseAggregateOp(parts[1])
	if err != nil {
		return cfg, fmt.Errorf("invalid op in spec %q: %w", spec, err)
	}
	cfg.Op = op
	if len(parts) == 3 {
		cfg.As = strings.TrimSpace(parts[2])
	}
	if cfg.SourceCol == "" && op != insyra.OpCountAll {
		return cfg, fmt.Errorf("invalid aggregate spec %q: source column is required for op %s", spec, op)
	}
	return cfg, nil
}

// parseAggregateOp accepts the canonical names plus a few well-known aliases
// (avg / std / population variants) used by the CLI DSL.
func parseAggregateOp(raw string) (insyra.AggregateOp, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "sum":
		return insyra.OpSum, nil
	case "mean", "avg":
		return insyra.OpMean, nil
	case "median":
		return insyra.OpMedian, nil
	case "min":
		return insyra.OpMin, nil
	case "max":
		return insyra.OpMax, nil
	case "count":
		return insyra.OpCount, nil
	case "countall":
		return insyra.OpCountAll, nil
	case "std", "stdev", "stddev":
		return insyra.OpStdev, nil
	case "stdp", "stdevp", "stddevp":
		return insyra.OpStdevP, nil
	case "var", "variance":
		return insyra.OpVar, nil
	case "varp":
		return insyra.OpVarP, nil
	case "first":
		return insyra.OpFirst, nil
	case "last":
		return insyra.OpLast, nil
	case "nunique":
		return insyra.OpNUnique, nil
	}
	return 0, fmt.Errorf("unknown aggregate op %q", raw)
}
