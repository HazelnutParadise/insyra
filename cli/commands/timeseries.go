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
	// `fillna` / `fillnan` are registered in fillna.go.

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
			"cov <other>              rolling sample covariance against DataList <other>",
			"beta <other>             rolling Cov(var, other) / Var(other)",
			"minobs <n>               minimum valid observations (default = window)",
			"center yes|no            anchor window at the central row (default no)",
		},
		Examples: []string{
			"insyra rolling price 3 mean as ma3",
			"insyra rolling price 7 mean minobs 1 as ma7_soft",
			"insyra rolling price 5 std center yes as roll_std",
			"insyra rolling asset 20 cov benchmark as roll_cov",
			"insyra rolling asset 20 beta benchmark minobs 10 as roll_beta",
		},
		Run: runRollingCommand,
	})
	_ = Register(&CommandHandler{
		Name:        "ewm",
		Usage:       "ewm <var> alpha|span|halflife <value> mean|var|std [adjust yes|no] [bias yes|no] [minobs <n>] [as <var>]",
		Description: "Exponentially weighted mean / variance / standard deviation over a DataList",
		Forms: []string{
			"alpha <value>            smoothing factor in (0, 1]",
			"span <value>             span >= 1, alpha = 2 / (span + 1)",
			"halflife <value>         half-life > 0, alpha = 1 - exp(ln(0.5) / halflife)",
			"<reducer>                mean, var, std",
			"adjust yes|no            use the adjusted (decaying-weight) form (default no)",
			"bias yes|no              population variance instead of the corrected one (default no)",
			"minobs <n>               minimum valid observations before emitting (default 1)",
		},
		Examples: []string{
			"insyra ewm price alpha 0.5 mean as ewma",
			"insyra ewm price span 12 mean adjust yes as ema12",
			"insyra ewm returns halflife 5 std minobs 3 as ewvol",
		},
		Run: runEWMCommand,
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

	// `cov` and `beta` take the next positional as the second series; option
	// parsing then continues from the token after it.
	optionStart := 3
	var other *insyra.DataList
	if reducer == "cov" || reducer == "beta" {
		if len(coreArgs) < 4 {
			return fmt.Errorf("rolling: reducer %q requires a second DataList variable (usage: rolling <var> <window> %s <other> [minobs <n>] [center yes|no] [as <var>])", reducer, reducer)
		}
		other, err = getDataListVar(ctx, coreArgs[3])
		if err != nil {
			return fmt.Errorf("rolling: %w", err)
		}
		optionStart = 4
	}

	opts := insyra.RollingOptions{Window: window}
	for i := optionStart; i < len(coreArgs); {
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
	case "cov":
		result = r.Cov(other)
	case "beta":
		result = r.Beta(other)
	default:
		return fmt.Errorf("rolling: unknown reducer %q (supported: sum, mean, min, max, median, std, var, cov <other>, beta <other>)", coreArgs[2])
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

func runEWMCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 4 {
		return fmt.Errorf("usage: ewm <var> alpha|span|halflife <value> mean|var|std [adjust yes|no] [bias yes|no] [minobs <n>] [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return fmt.Errorf("ewm: %w", err)
	}

	decay := strings.ToLower(coreArgs[1])
	value, err := strconv.ParseFloat(coreArgs[2], 64)
	if err != nil {
		return fmt.Errorf("ewm: invalid %s %q: %w", decay, coreArgs[2], err)
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("ewm: %s must be a finite number, got %q", decay, coreArgs[2])
	}

	var opts insyra.EWMOptions
	switch decay {
	case "alpha":
		if value <= 0 || value > 1 {
			return fmt.Errorf("ewm: alpha must be in (0, 1], got %v", value)
		}
		opts.Alpha = value
	case "span":
		if value < 1 {
			return fmt.Errorf("ewm: span must be >= 1, got %v", value)
		}
		opts.Span = value
	case "halflife":
		if value <= 0 {
			return fmt.Errorf("ewm: halflife must be > 0, got %v", value)
		}
		opts.HalfLife = value
	default:
		return fmt.Errorf("ewm: unknown decay keyword %q (supported: alpha, span, halflife)", coreArgs[1])
	}

	reducer := strings.ToLower(coreArgs[3])
	for i := 4; i < len(coreArgs); {
		key := strings.ToLower(coreArgs[i])
		if i+1 >= len(coreArgs) {
			return fmt.Errorf("ewm: option %q requires a value", coreArgs[i])
		}
		raw := coreArgs[i+1]
		switch key {
		case "adjust":
			b, err := parseFlexBool(raw)
			if err != nil {
				return fmt.Errorf("ewm: invalid value for adjust: %w", err)
			}
			opts.Adjust = b
		case "bias":
			b, err := parseFlexBool(raw)
			if err != nil {
				return fmt.Errorf("ewm: invalid value for bias: %w", err)
			}
			opts.Bias = b
		case "minobs":
			n, err := strconv.Atoi(raw)
			if err != nil {
				return fmt.Errorf("ewm: invalid minobs %q: %w", raw, err)
			}
			opts.MinObs = n
		default:
			return fmt.Errorf("ewm: unknown option %q (supported: adjust, bias, minobs)", coreArgs[i])
		}
		i += 2
	}

	e := dl.Clone().EWM(opts)
	var result *insyra.DataList
	switch reducer {
	case "mean":
		result = e.Mean()
	case "var":
		result = e.Var()
	case "std", "stdev":
		result = e.Std()
	default:
		return fmt.Errorf("ewm: unknown reducer %q (supported: mean, var, std)", coreArgs[3])
	}
	if result.Len() != dl.Len() {
		return fmt.Errorf("ewm: invalid options (alpha %v, span %v, halflife %v)", opts.Alpha, opts.Span, opts.HalfLife)
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}
