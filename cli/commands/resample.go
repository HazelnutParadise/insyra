package commands

import (
	"fmt"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

// resampleOps is the operator list shown when a spec names an unknown op. It
// mirrors parseAggregateOp (shared with `groupby`) so the two commands stay
// identical in what they accept.
const resampleOps = "sum, mean (avg), median, min, max, count, countall, std (stdev), stdp (stdevp), var, varp, first, last, nunique"

func init() {
	_ = Register(&CommandHandler{
		Name:        "resample",
		Usage:       "resample <dt> <timecol> weekly|monthly|quarterly|yearly <col>:<op>[:<name>] [<col>:<op>[:<name>] ...] [as <var>]",
		Description: "Aggregate a time-indexed DataTable into calendar periods",
		Forms: []string{
			"<timecol>                column of time.Time values keying every row",
			"<frequency>              weekly, monthly, quarterly, yearly",
			"<col>:<op>[:<name>]      one aggregate per output column; <name> renames it",
			"",
			"Ops: " + resampleOps,
		},
		Examples: []string{
			"insyra resample bars Date monthly Open:first High:max Low:min Close:last Volume:sum as monthly_bars",
			"insyra resample bars Date weekly Close:last:WeekClose as weekly_close",
			"insyra resample sales Date quarterly revenue:sum:total",
		},
		Run: runResampleCommand,
	})
}

func runResampleCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 4 {
		return fmt.Errorf("usage: resample <dt> <timecol> weekly|monthly|quarterly|yearly <col>:<op>[:<name>] [...] [as <var>]")
	}
	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return fmt.Errorf("resample: %w", err)
	}
	timeCol := coreArgs[1]
	freq, err := parseResampleFreq(coreArgs[2])
	if err != nil {
		return err
	}

	aggs := make([]insyra.ResampleAgg, 0, len(coreArgs)-3)
	for _, spec := range coreArgs[3:] {
		agg, err := parseResampleSpec(spec)
		if err != nil {
			return err
		}
		aggs = append(aggs, agg)
	}

	result, err := table.Resample(timeCol, freq, aggs...)
	if err != nil {
		return fmt.Errorf("resample: %w", err)
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "resampled into %s (%d rows)\n", alias, result.NumRows())
	return nil
}

func parseResampleFreq(raw string) (insyra.ResampleFreq, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "weekly":
		return insyra.ResampleWeekly, nil
	case "monthly":
		return insyra.ResampleMonthly, nil
	case "quarterly":
		return insyra.ResampleQuarterly, nil
	case "yearly":
		return insyra.ResampleYearly, nil
	}
	return 0, fmt.Errorf("resample: unknown frequency %q (supported: weekly, monthly, quarterly, yearly)", raw)
}

// parseResampleSpec parses one `<col>:<op>[:<name>]` descriptor. Column names
// containing ':' cannot be expressed in this syntax.
func parseResampleSpec(spec string) (insyra.ResampleAgg, error) {
	var agg insyra.ResampleAgg
	parts := strings.Split(spec, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return agg, fmt.Errorf("resample: invalid spec %q (expected <col>:<op>[:<name>])", spec)
	}
	agg.Col = strings.TrimSpace(parts[0])
	if agg.Col == "" {
		return agg, fmt.Errorf("resample: invalid spec %q (expected <col>:<op>[:<name>]): source column is required", spec)
	}
	op, err := parseAggregateOp(parts[1])
	if err != nil {
		return agg, fmt.Errorf("resample: invalid op in spec %q: %w (supported: %s)", spec, err, resampleOps)
	}
	agg.Op = op
	if len(parts) == 3 {
		agg.As = strings.TrimSpace(parts[2])
		if agg.As == "" {
			return agg, fmt.Errorf("resample: invalid spec %q (expected <col>:<op>[:<name>]): output name is empty", spec)
		}
	}
	return agg, nil
}
