package commands

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{Name: "movavg", Usage: "movavg <var> <window> [as <var>]", Description: "Moving average", Run: runMovAvgCommand})
	_ = Register(&CommandHandler{Name: "expsmooth", Usage: "expsmooth <var> <alpha> [as <var>]", Description: "Exponential smoothing", Run: runExpSmoothCommand})
	_ = Register(&CommandHandler{Name: "diff", Usage: "diff <var> [as <var>]", Description: "Difference (legacy, length n-1)", Run: runDiffCommand})
	_ = Register(&CommandHandler{
		Name:        "fillna",
		Usage:       "fillna <var> mean|median|mode|ffill|bfill|interpolate [cols A,B,C] [limit N] [extrapolate yes|no] [missing nan|nil|both] [as <var>]",
		Description: "Fill missing DataList/DataTable values",
		Forms: []string{
			"mean|median|interpolate  apply to numeric data only",
			"mode|ffill|bfill         work with any data type",
			"cols <A,B,C>             DataTable only; comma-separated column names or indices",
			"limit <n>                cap consecutive ffill/bfill replacements (0 = unlimited)",
			"extrapolate yes|no       interpolation only; fill leading/trailing gaps (default no)",
			"missing nan|nil|both     which kind of missing to fill (default both)",
		},
		Examples: []string{
			"insyra fillna price median as price_filled",
			"insyra fillna price ffill limit 2 missing nan as price_ffill",
			"insyra fillna price interpolate extrapolate yes as price_interp",
			"insyra fillna sales median cols revenue,cost as cleaned",
		},
		Run: runFillNACommand,
	})
	_ = Register(&CommandHandler{
		Name:        "fillnan",
		Usage:       "fillnan <var> mean [as <var>]",
		Description: "Fill NaN with mean (deprecated alias; prefer 'fillna ... missing nan')",
		Run:         runFillNaNCommand,
	})

	_ = Register(&CommandHandler{
		Name:        "shift",
		Usage:       "shift <var> <periods> [fill <value>] [as <var>]",
		Description: "Shift / lag / lead a DataList (positive = lag, negative = lead)",
		Forms: []string{
			"<periods>                positive shifts right (lag), negative shifts left (lead)",
			"fill <value>             value to put in empty slots (default nil)",
		},
		Examples: []string{
			"insyra shift price 1 as prev_price",
			"insyra shift price -1 as next_price",
			"insyra shift price 2 fill 0 as p_shifted",
		},
		Run: runShiftCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "diffn",
		Usage:       "diffn <var> <periods> [as <var>]",
		Description: "Backward difference, same-length output with leading nils (use diff for legacy length-n-1 behaviour)",
		Examples: []string{
			"insyra diffn price 1 as d1",
			"insyra diffn price 7 as weekly_delta",
		},
		Run: runDiffNCommand,
	})
	_ = Register(&CommandHandler{Name: "pctchange", Usage: "pctchange <var> <periods> [as <var>]", Description: "Percent change over `periods` rows", Run: runPctChangeCommand})
	_ = Register(&CommandHandler{Name: "cumsum", Usage: "cumsum <var> [as <var>]", Description: "Running total", Run: runCumSumCommand})
	_ = Register(&CommandHandler{Name: "cumprod", Usage: "cumprod <var> [as <var>]", Description: "Running product", Run: runCumProdCommand})
	_ = Register(&CommandHandler{Name: "cummax", Usage: "cummax <var> [as <var>]", Description: "Running maximum (historical high)", Run: runCumMaxCommand})
	_ = Register(&CommandHandler{Name: "cummin", Usage: "cummin <var> [as <var>]", Description: "Running minimum (historical low)", Run: runCumMinCommand})

	_ = Register(&CommandHandler{
		Name:        "rolling",
		Usage:       "rolling <var> <window> <reducer> [minobs <n>] [center yes|no] [as <var>]",
		Description: "Rolling-window reduction over a DataList",
		Forms: []string{
			"<reducer>                sum, mean, min, max, median, std, var",
			"minobs <n>               minimum valid observations (default = window)",
			"center yes|no            anchor window at the central row (default no)",
		},
		Examples: []string{
			"insyra rolling price 3 mean as ma3",
			"insyra rolling price 7 mean minobs 1 as ma7_soft",
			"insyra rolling price 5 std center yes as roll_std",
		},
		Run: runRollingCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "expanding",
		Usage:       "expanding <var> <minobs> <reducer> [as <var>]",
		Description: "Expanding-window reduction (in[0..=i]) over a DataList",
		Forms: []string{
			"<reducer>                sum, mean, min, max, median, std, var",
			"<minobs>                 minimum valid observations before emitting (>=1)",
		},
		Examples: []string{
			"insyra expanding price 1 mean as emean",
			"insyra expanding pnl 5 sum as cumulative_pnl",
		},
		Run: runExpandingCommand,
	})
}

func runMovAvgCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: movavg <var> <window> [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	window, err := strconv.Atoi(coreArgs[1])
	if err != nil {
		return err
	}
	ctx.Vars[alias] = dl.Clone().MovingAverage(window)
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runExpSmoothCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: expsmooth <var> <alpha> [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	alpha, err := strconv.ParseFloat(coreArgs[1], 64)
	if err != nil {
		return err
	}
	ctx.Vars[alias] = dl.Clone().ExponentialSmoothing(alpha)
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runDiffCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: diff <var> [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	ctx.Vars[alias] = dl.Clone().Difference()
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

// runFillNaNCommand handles the legacy `fillnan <var> mean [as <var>]` shape.
// It only fills NaN (not nil) and only supports the mean strategy. Use `fillna`
// for the full surface.
func runFillNaNCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 || strings.ToLower(coreArgs[1]) != "mean" {
		return fmt.Errorf("usage: fillnan <var> mean [as <var>] (use 'fillna' for other strategies)")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	if len(coreArgs) > 2 {
		return fmt.Errorf("fillnan: unexpected extra args %v (use 'fillna' for options)", coreArgs[2:])
	}
	result := dl.Clone().FillNaNWithMean()
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "warning: 'fillnan' is deprecated; use 'fillna %s mean missing nan%s' instead\n", coreArgs[0], aliasSuffix(alias))
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func aliasSuffix(alias string) string {
	if alias == "" || alias == "$result" {
		return ""
	}
	return " as " + alias
}

type fillNAOptions struct {
	Cols        []string
	Limit       int
	Extrapolate bool
	Missing     string // "nan", "nil", or "both" (default)
}

func runFillNACommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: fillna <var> mean|median|mode|ffill|bfill|interpolate [cols A,B,C] [limit N] [extrapolate yes|no] [missing nan|nil|both] [as <var>]")
	}
	strategy := strings.ToLower(coreArgs[1])
	opts, err := parseFillNAOptions(coreArgs[2:])
	if err != nil {
		return err
	}

	if dt, err := getDataTableVar(ctx, coreArgs[0]); err == nil {
		result, err := applyFillNAToTable(dt, strategy, opts)
		if err != nil {
			return err
		}
		ctx.Vars[alias] = result
		_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
		return nil
	}
	if dl, err := getDataListVar(ctx, coreArgs[0]); err == nil {
		if len(opts.Cols) > 0 {
			return fmt.Errorf("fillna: 'cols' only applies to DataTable variables")
		}
		result, err := applyFillNAToList(dl, strategy, opts)
		if err != nil {
			return err
		}
		ctx.Vars[alias] = result
		_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
		return nil
	}
	return fmt.Errorf("variable not found: %s", coreArgs[0])
}

func applyFillNAToList(dl *insyra.DataList, strategy string, opts fillNAOptions) (*insyra.DataList, error) {
	result := dl.Clone()
	preserved := snapshotPreservedDL(result, opts.Missing)
	if err := runListStrategy(result, strategy, opts); err != nil {
		return nil, err
	}
	restorePreservedDL(result, opts.Missing, preserved)
	return result, nil
}

func applyFillNAToTable(dt *insyra.DataTable, strategy string, opts fillNAOptions) (*insyra.DataTable, error) {
	result := dt.Clone()
	preserved := snapshotPreservedDT(result, opts.Cols, opts.Missing)
	if err := runTableStrategy(result, strategy, opts); err != nil {
		return nil, err
	}
	restorePreservedDT(result, opts.Missing, preserved)
	return result, nil
}

func runListStrategy(dl *insyra.DataList, strategy string, opts fillNAOptions) error {
	switch strategy {
	case "mean":
		dl.FillWithMean()
	case "median":
		dl.FillWithMedian()
	case "mode":
		dl.FillWithMode()
	case "ffill":
		dl.FillForward(opts.Limit)
	case "bfill":
		dl.FillBackward(opts.Limit)
	case "interpolate":
		dl.FillByInterpolation(opts.Extrapolate)
	default:
		return fmt.Errorf("fillna: unknown strategy %q (supported: mean, median, mode, ffill, bfill, interpolate)", strategy)
	}
	return nil
}

func runTableStrategy(dt *insyra.DataTable, strategy string, opts fillNAOptions) error {
	switch strategy {
	case "mean":
		dt.FillWithMean(opts.Cols...)
	case "median":
		dt.FillWithMedian(opts.Cols...)
	case "mode":
		dt.FillWithMode(opts.Cols...)
	case "ffill":
		dt.FillForward(opts.Limit, opts.Cols...)
	case "bfill":
		dt.FillBackward(opts.Limit, opts.Cols...)
	case "interpolate":
		dt.FillByInterpolation(opts.Cols...)
	default:
		return fmt.Errorf("fillna: unknown strategy %q (supported: mean, median, mode, ffill, bfill, interpolate)", strategy)
	}
	return nil
}

// snapshotPreservedDL records positions whose original value must be restored
// after the fill runs. With missing=="nan", nil positions are preserved (so the
// fill only sticks at NaN positions). With missing=="nil", NaN positions are
// preserved. With "both" (or unset), nothing is preserved.
func snapshotPreservedDL(dl *insyra.DataList, missing string) []int {
	if missing != "nan" && missing != "nil" {
		return nil
	}
	var positions []int
	for i, v := range dl.Data() {
		if missing == "nan" && v == nil {
			positions = append(positions, i)
		} else if missing == "nil" {
			if f, ok := v.(float64); ok && math.IsNaN(f) {
				positions = append(positions, i)
			}
		}
	}
	return positions
}

func restorePreservedDL(dl *insyra.DataList, missing string, positions []int) {
	if len(positions) == 0 {
		return
	}
	var restoreVal any
	if missing == "nan" {
		restoreVal = nil
	} else {
		restoreVal = math.NaN()
	}
	for _, i := range positions {
		dl.Update(i, restoreVal)
	}
}

func snapshotPreservedDT(dt *insyra.DataTable, cols []string, missing string) map[string][]int {
	if missing != "nan" && missing != "nil" {
		return nil
	}
	targets := cols
	if len(targets) == 0 {
		targets = dt.ColNames()
	}
	preserved := map[string][]int{}
	for _, col := range targets {
		dl := dt.GetColByName(col)
		if dl == nil {
			continue
		}
		positions := snapshotPreservedDL(dl, missing)
		if len(positions) > 0 {
			preserved[col] = positions
		}
	}
	return preserved
}

func restorePreservedDT(dt *insyra.DataTable, missing string, preserved map[string][]int) {
	if len(preserved) == 0 {
		return
	}
	for col, positions := range preserved {
		dl := dt.GetColByName(col)
		if dl == nil {
			continue
		}
		restorePreservedDL(dl, missing, positions)
	}
}

func parseFillNAOptions(args []string) (fillNAOptions, error) {
	opts := fillNAOptions{Missing: "both"}
	for i := 0; i < len(args); {
		key := strings.ToLower(args[i])
		next := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("fillna: option %q requires a value", args[i])
			}
			return args[i+1], nil
		}
		switch key {
		case "cols":
			v, err := next()
			if err != nil {
				return opts, err
			}
			opts.Cols = parseCSVTokens(v)
			i += 2
		case "limit":
			v, err := next()
			if err != nil {
				return opts, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return opts, fmt.Errorf("fillna: invalid limit %q", v)
			}
			opts.Limit = n
			i += 2
		case "extrapolate":
			v, err := next()
			if err != nil {
				return opts, err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return opts, fmt.Errorf("fillna: invalid value for extrapolate: %w", err)
			}
			opts.Extrapolate = b
			i += 2
		case "missing":
			v, err := next()
			if err != nil {
				return opts, err
			}
			switch strings.ToLower(v) {
			case "nan", "nil", "both":
				opts.Missing = strings.ToLower(v)
			default:
				return opts, fmt.Errorf("fillna: invalid value for missing %q (supported: nan, nil, both)", v)
			}
			i += 2
		default:
			return opts, fmt.Errorf("fillna: unknown option %q (supported: cols, limit, extrapolate, missing)", args[i])
		}
	}
	return opts, nil
}

// ===========================================================================
// Window / sequence transforms (Issue #162)
// ===========================================================================

func runShiftCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: shift <var> <periods> [fill <value>] [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	periods, err := strconv.Atoi(coreArgs[1])
	if err != nil {
		return fmt.Errorf("shift: invalid periods %q: %w", coreArgs[1], err)
	}
	var fillArgs []any
	for i := 2; i < len(coreArgs); {
		if strings.EqualFold(coreArgs[i], "fill") {
			if i+1 >= len(coreArgs) {
				return fmt.Errorf("shift: option %q requires a value", coreArgs[i])
			}
			fillArgs = []any{parseLiteral(coreArgs[i+1])}
			i += 2
			continue
		}
		return fmt.Errorf("shift: unknown option %q (supported: fill)", coreArgs[i])
	}
	ctx.Vars[alias] = dl.Clone().Shift(periods, fillArgs...)
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runDiffNCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: diffn <var> <periods> [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	periods, err := strconv.Atoi(coreArgs[1])
	if err != nil {
		return fmt.Errorf("diffn: invalid periods %q: %w", coreArgs[1], err)
	}
	result := dl.Clone().Diff(periods)
	if result == nil {
		return fmt.Errorf("diffn: periods must be > 0")
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runPctChangeCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: pctchange <var> <periods> [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	periods, err := strconv.Atoi(coreArgs[1])
	if err != nil {
		return fmt.Errorf("pctchange: invalid periods %q: %w", coreArgs[1], err)
	}
	result := dl.Clone().PctChange(periods)
	if result == nil {
		return fmt.Errorf("pctchange: periods must be > 0")
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runCumSumCommand(ctx *ExecContext, args []string) error {
	return runCumulative(ctx, args, "cumsum", func(dl *insyra.DataList) *insyra.DataList { return dl.CumSum() })
}

func runCumProdCommand(ctx *ExecContext, args []string) error {
	return runCumulative(ctx, args, "cumprod", func(dl *insyra.DataList) *insyra.DataList { return dl.CumProd() })
}

func runCumMaxCommand(ctx *ExecContext, args []string) error {
	return runCumulative(ctx, args, "cummax", func(dl *insyra.DataList) *insyra.DataList { return dl.CumMax() })
}

func runCumMinCommand(ctx *ExecContext, args []string) error {
	return runCumulative(ctx, args, "cummin", func(dl *insyra.DataList) *insyra.DataList { return dl.CumMin() })
}

func runCumulative(ctx *ExecContext, args []string, name string, fn func(*insyra.DataList) *insyra.DataList) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: %s <var> [as <var>]", name)
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	ctx.Vars[alias] = fn(dl.Clone())
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runRollingCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 3 {
		return fmt.Errorf("usage: rolling <var> <window> <reducer> [minobs <n>] [center yes|no] [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	window, err := strconv.Atoi(coreArgs[1])
	if err != nil {
		return fmt.Errorf("rolling: invalid window %q: %w", coreArgs[1], err)
	}
	reducer := strings.ToLower(coreArgs[2])

	opts := insyra.RollingOptions{Window: window}
	for i := 3; i < len(coreArgs); {
		key := strings.ToLower(coreArgs[i])
		next := func() (string, error) {
			if i+1 >= len(coreArgs) {
				return "", fmt.Errorf("rolling: option %q requires a value", coreArgs[i])
			}
			return coreArgs[i+1], nil
		}
		switch key {
		case "minobs":
			v, err := next()
			if err != nil {
				return err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("rolling: invalid minobs %q: %w", v, err)
			}
			opts.MinObs = n
			i += 2
		case "center":
			v, err := next()
			if err != nil {
				return err
			}
			b, err := parseFlexBool(v)
			if err != nil {
				return fmt.Errorf("rolling: invalid value for center: %w", err)
			}
			opts.Center = b
			i += 2
		default:
			return fmt.Errorf("rolling: unknown option %q (supported: minobs, center)", coreArgs[i])
		}
	}

	r := dl.Clone().Rolling(opts)
	var result *insyra.DataList
	switch reducer {
	case "sum":
		result = r.Sum()
	case "mean", "avg":
		result = r.Mean()
	case "min":
		result = r.Min()
	case "max":
		result = r.Max()
	case "median":
		result = r.Median()
	case "std", "stdev":
		result = r.Std()
	case "var":
		result = r.Var()
	default:
		return fmt.Errorf("rolling: unknown reducer %q (supported: sum, mean, min, max, median, std, var)", coreArgs[2])
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runExpandingCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 3 {
		return fmt.Errorf("usage: expanding <var> <minobs> <reducer> [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	minObs, err := strconv.Atoi(coreArgs[1])
	if err != nil {
		return fmt.Errorf("expanding: invalid minobs %q: %w", coreArgs[1], err)
	}
	reducer := strings.ToLower(coreArgs[2])

	e := dl.Clone().Expanding(minObs)
	var result *insyra.DataList
	switch reducer {
	case "sum":
		result = e.Sum()
	case "mean", "avg":
		result = e.Mean()
	case "min":
		result = e.Min()
	case "max":
		result = e.Max()
	case "median":
		result = e.Median()
	case "std", "stdev":
		result = e.Std()
	case "var":
		result = e.Var()
	default:
		return fmt.Errorf("expanding: unknown reducer %q (supported: sum, mean, min, max, median, std, var)", coreArgs[2])
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}
