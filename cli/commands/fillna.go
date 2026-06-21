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
	// The deprecated `fillnan` command intentionally keeps NaN-only mean fill.
	result := dl.Clone().FillNaNWithMean() // nolint:staticcheck
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
